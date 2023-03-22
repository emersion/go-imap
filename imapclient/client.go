// Package imapclient implements an IMAP client.
package imapclient

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const dateTimeLayout = "_2-Jan-2006 15:04:05 -0700"

const (
	idleReadTimeout    = time.Duration(0)
	respReadTimeout    = 30 * time.Second
	literalReadTimeout = 5 * time.Minute

	respWriteTimeout    = 30 * time.Second
	literalWriteTimeout = 5 * time.Minute
)

// State describes the client state.
//
// See RFC 9051 section 3.
type State int

const (
	StateNone State = iota
	StateNotAuthenticated
	StateAuthenticated
	StateSelected
	StateLogout
)

// String implements fmt.Stringer.
func (state State) String() string {
	switch state {
	case StateNone:
		return "none"
	case StateNotAuthenticated:
		return "not authenticated"
	case StateAuthenticated:
		return "authenticated"
	case StateSelected:
		return "selected"
	case StateLogout:
		return "logout"
	default:
		panic(fmt.Errorf("imapclient: unknown state %v", int(state)))
	}
}

// SelectedMailbox contains metadata for the currently selected mailbox.
type SelectedMailbox struct {
	Name           string
	NumMessages    uint32
	Flags          []imap.Flag
	PermanentFlags []imap.Flag
}

func (mbox *SelectedMailbox) copy() *SelectedMailbox {
	copy := *mbox
	return &copy
}

// Options contains options for Client.
type Options struct {
	// Raw ingress and egress data will be written to this writer, if any
	DebugWriter io.Writer
	// Unilateral data handler
	UnilateralDataHandler *UnilateralDataHandler
	// Decoder for RFC 2047 words
	WordDecoder *mime.WordDecoder
}

func (options *Options) wrapReadWriter(rw io.ReadWriter) io.ReadWriter {
	if options.DebugWriter == nil {
		return rw
	}
	return struct {
		io.Reader
		io.Writer
	}{
		Reader: io.TeeReader(rw, options.DebugWriter),
		Writer: io.MultiWriter(rw, options.DebugWriter),
	}
}

func (options *Options) decodeText(s string) (string, error) {
	wordDecoder := options.WordDecoder
	if wordDecoder == nil {
		wordDecoder = &mime.WordDecoder{}
	}
	out, err := wordDecoder.DecodeHeader(s)
	if err != nil {
		return s, err
	}
	return out, nil
}

func (options *Options) unilateralDataHandler() *UnilateralDataHandler {
	if options.UnilateralDataHandler == nil {
		return &UnilateralDataHandler{}
	}
	return options.UnilateralDataHandler
}

// Client is an IMAP client.
//
// IMAP commands are exposed as methods. These methods will block until the
// command has been sent to the server, but won't block until the server sends
// a response. They return a command struct which can be used to wait for the
// server response. This can be used to execute multiple commands concurrently,
// however care must be taken to avoid ambiguities. See RFC 9051 section 5.5.
//
// A client can be safely used from multiple goroutines, however this doesn't
// guarantee any command ordering and is subject to the same caveats as command
// pipelining (see above). Additionally, some commands (e.g. StartTLS,
// Authenticate, Idle) block the client during their execution.
type Client struct {
	conn     net.Conn
	options  Options
	br       *bufio.Reader
	bw       *bufio.Writer
	dec      *imapwire.Decoder
	encMutex sync.Mutex

	greetingCh   chan struct{}
	greetingRecv bool
	greetingErr  error

	decCh  chan struct{}
	decErr error

	mutex       sync.Mutex
	state       State
	caps        imap.CapSet
	mailbox     *SelectedMailbox
	cmdTag      uint64
	pendingCmds []command
	contReqs    []continuationRequest
	closed      bool
}

// New creates a new IMAP client.
//
// This function doesn't perform I/O.
//
// A nil options pointer is equivalent to a zero options value.
func New(conn net.Conn, options *Options) *Client {
	if options == nil {
		options = &Options{}
	}

	rw := options.wrapReadWriter(conn)
	br := bufio.NewReader(rw)
	bw := bufio.NewWriter(rw)

	client := &Client{
		conn:       conn,
		options:    *options,
		br:         br,
		bw:         bw,
		dec:        imapwire.NewDecoder(br),
		greetingCh: make(chan struct{}),
		decCh:      make(chan struct{}),
		state:      StateNone,
	}
	go client.read()
	return client
}

// DialTLS connects to an IMAP server with implicit TLS.
func DialTLS(address string, options *Options) (*Client, error) {
	conn, err := tls.Dial("tcp", address, &tls.Config{
		NextProtos: []string{"imap"},
	})
	if err != nil {
		return nil, err
	}
	return New(conn, options), nil
}

// DialStartTLS connects to an IMAP server with STARTTLS.
func DialStartTLS(address string, options *Options) (*Client, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	client := New(conn, options)
	if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
		conn.Close()
		return nil, err
	}

	return client, err
}

func (c *Client) setReadTimeout(dur time.Duration) {
	if dur > 0 {
		c.conn.SetReadDeadline(time.Now().Add(dur))
	} else {
		c.conn.SetReadDeadline(time.Time{})
	}
}

func (c *Client) setWriteTimeout(dur time.Duration) {
	if dur > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(dur))
	} else {
		c.conn.SetWriteDeadline(time.Time{})
	}
}

// State returns the current state of the client.
func (c *Client) State() State {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.state
}

func (c *Client) setState(state State) {
	c.mutex.Lock()
	c.state = state
	if c.state != StateSelected {
		c.mailbox = nil
	}
	c.mutex.Unlock()
}

// Caps returns the capabilities advertised by the server.
//
// When the server hasn't sent the capability list, this method will request it
// and block until it's received. If the capabilities cannot be fetched, nil is
// returned.
func (c *Client) Caps() imap.CapSet {
	if err := c.WaitGreeting(); err != nil {
		return nil
	}

	c.mutex.Lock()
	caps := c.caps
	c.mutex.Unlock()

	if caps != nil {
		return caps
	}

	caps, _ = c.Capability().Wait()
	return caps
}

func (c *Client) setCaps(caps imap.CapSet) {
	c.mutex.Lock()
	c.caps = caps
	c.mutex.Unlock()
}

// Mailbox returns the state of the currently selected mailbox.
//
// If there is no currently selected mailbox, nil is returned.
//
// The returned struct must not be mutated.
func (c *Client) Mailbox() *SelectedMailbox {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.mailbox
}

// Close immediately closes the connection.
func (c *Client) Close() error {
	c.mutex.Lock()
	alreadyClosed := c.closed
	c.closed = true
	c.mutex.Unlock()

	// Ignore net.ErrClosed here, because we also call conn.Close in c.read
	if err := c.conn.Close(); err != nil && err != net.ErrClosed {
		return err
	}

	<-c.decCh
	if err := c.decErr; err != nil {
		return err
	}

	if alreadyClosed {
		return net.ErrClosed
	}
	return nil
}

// beginCommand starts sending a command to the server.
//
// The command name and a space are written.
//
// The caller must call commandEncoder.end.
func (c *Client) beginCommand(name string, cmd command) *commandEncoder {
	c.encMutex.Lock() // unlocked by commandEncoder.end

	c.mutex.Lock()
	c.cmdTag++
	tag := fmt.Sprintf("T%v", c.cmdTag)
	c.pendingCmds = append(c.pendingCmds, cmd)
	c.mutex.Unlock()

	c.setWriteTimeout(respWriteTimeout)

	baseCmd := cmd.base()
	*baseCmd = Command{
		tag:  tag,
		done: make(chan error, 1),
	}
	enc := &commandEncoder{
		Encoder: imapwire.NewEncoder(c.bw),
		client:  c,
		cmd:     baseCmd,
	}
	enc.Atom(tag).SP().Atom(name)
	return enc
}

func (c *Client) deletePendingCmdByTag(tag string) command {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for i, cmd := range c.pendingCmds {
		if cmd.base().tag == tag {
			c.pendingCmds = append(c.pendingCmds[:i], c.pendingCmds[i+1:]...)
			return cmd
		}
	}
	return nil
}

func (c *Client) findPendingCmdFunc(f func(cmd command) bool) command {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, cmd := range c.pendingCmds {
		if f(cmd) {
			return cmd
		}
	}
	return nil
}

func findPendingCmdByType[T command](c *Client) T {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, cmd := range c.pendingCmds {
		if cmd, ok := cmd.(T); ok {
			return cmd
		}
	}

	var cmd T
	return cmd
}

func (c *Client) completeCommand(cmd command, err error) {
	done := cmd.base().done
	done <- err
	close(done)

	// Ensure the command is not blocked waiting on continuation requests
	c.mutex.Lock()
	var filtered []continuationRequest
	for _, contReq := range c.contReqs {
		if contReq.cmd != cmd.base() {
			filtered = append(filtered, contReq)
		} else {
			contReq.Cancel(err)
		}
	}
	c.contReqs = filtered
	c.mutex.Unlock()

	switch cmd := cmd.(type) {
	case *authenticateCommand, *loginCommand:
		if err == nil {
			c.setState(StateAuthenticated)
		}
	case *SelectCommand:
		if err == nil {
			c.mutex.Lock()
			c.state = StateSelected
			c.mailbox = &SelectedMailbox{
				Name:           cmd.mailbox,
				NumMessages:    cmd.data.NumMessages,
				Flags:          cmd.data.Flags,
				PermanentFlags: cmd.data.PermanentFlags,
			}
			c.mutex.Unlock()
		}
	case *unselectCommand:
		if err == nil {
			c.setState(StateAuthenticated)
		}
	case *logoutCommand:
		if err == nil {
			c.setState(StateLogout)
		}
	case *ListCommand:
		if cmd.pendingData != nil {
			cmd.mailboxes <- cmd.pendingData
		}
		close(cmd.mailboxes)
	case *FetchCommand:
		close(cmd.msgs)
	case *ExpungeCommand:
		close(cmd.seqNums)
	}
}

func (c *Client) registerContReq(cmd command) *imapwire.ContinuationRequest {
	contReq := imapwire.NewContinuationRequest()

	c.mutex.Lock()
	c.contReqs = append(c.contReqs, continuationRequest{
		ContinuationRequest: contReq,
		cmd:                 cmd.base(),
	})
	c.mutex.Unlock()

	return contReq
}

func (c *Client) unregisterContReq(contReq *imapwire.ContinuationRequest) {
	c.mutex.Lock()
	for i := range c.contReqs {
		if c.contReqs[i].ContinuationRequest == contReq {
			c.contReqs = append(c.contReqs[:i], c.contReqs[i+1:]...)
			break
		}
	}
	c.mutex.Unlock()
}

// read continuously reads data coming from the server.
//
// All the data is decoded in the read goroutine, then dispatched via channels
// to pending commands.
func (c *Client) read() {
	defer close(c.decCh)
	defer func() {
		if v := recover(); v != nil {
			c.decErr = fmt.Errorf("imapclient: panic reading response: %v\n%s", v, debug.Stack())
		}

		c.conn.Close()

		c.mutex.Lock()
		c.state = StateLogout
		pendingCmds := c.pendingCmds
		c.pendingCmds = nil
		c.mutex.Unlock()

		cmdErr := c.decErr
		if cmdErr == nil {
			cmdErr = io.ErrUnexpectedEOF
		}
		for _, cmd := range pendingCmds {
			c.completeCommand(cmd, cmdErr)
		}
	}()

	c.setReadTimeout(idleReadTimeout)
	for {
		if c.dec.EOF() {
			break
		}
		if err := c.readResponse(); err != nil {
			c.decErr = err
			break
		}
		if c.greetingErr != nil {
			break
		}
	}
}

func (c *Client) readResponse() error {
	c.setReadTimeout(respReadTimeout)
	defer c.setReadTimeout(idleReadTimeout)

	if c.dec.Special('+') {
		if err := c.readContinueReq(); err != nil {
			return fmt.Errorf("in continue-req: %v", err)
		}
		return nil
	}

	var tag, typ string
	if !c.dec.Expect(c.dec.Special('*') || c.dec.Atom(&tag), "'*' or atom") {
		return fmt.Errorf("in response: cannot read tag: %v", c.dec.Err())
	}
	if !c.dec.ExpectSP() {
		return fmt.Errorf("in response: %v", c.dec.Err())
	}
	if !c.dec.ExpectAtom(&typ) {
		return fmt.Errorf("in response: cannot read type: %v", c.dec.Err())
	}

	var (
		token    string
		err      error
		startTLS *startTLSCommand
	)
	if tag != "" {
		token = "response-tagged"
		startTLS, err = c.readResponseTagged(tag, typ)
	} else if typ == "BYE" {
		token = "resp-cond-bye"
		var text string
		if !c.dec.ExpectText(&text) {
			return fmt.Errorf("in resp-text: %v", c.dec.Err())
		}
	} else {
		token = "response-data"
		err = c.readResponseData(typ)
	}
	if err != nil {
		return fmt.Errorf("in %v: %v", token, err)
	}

	if !c.dec.ExpectCRLF() {
		return fmt.Errorf("in response: %v", c.dec.Err())
	}

	if startTLS != nil {
		c.upgradeStartTLS(startTLS.tlsConfig)
		close(startTLS.upgradeDone)
	}

	return nil
}

func (c *Client) readContinueReq() error {
	var text string
	if c.dec.SP() {
		c.dec.Text(&text)
	}
	if !c.dec.ExpectCRLF() {
		return c.dec.Err()
	}

	var contReq *imapwire.ContinuationRequest
	c.mutex.Lock()
	if len(c.contReqs) > 0 {
		contReq = c.contReqs[0].ContinuationRequest
		c.contReqs = append(c.contReqs[:0], c.contReqs[1:]...)
	}
	c.mutex.Unlock()

	if contReq == nil {
		return fmt.Errorf("received unmatched continuation request")
	}

	contReq.Done(text)
	return nil
}

func (c *Client) readResponseTagged(tag, typ string) (*startTLSCommand, error) {
	cmd := c.deletePendingCmdByTag(tag)
	if cmd == nil {
		return nil, fmt.Errorf("received tagged response with unknown tag %q", tag)
	}

	if !c.dec.ExpectSP() {
		return nil, c.dec.Err()
	}
	var code string
	if c.dec.Special('[') { // resp-text-code
		if !c.dec.ExpectAtom(&code) {
			return nil, fmt.Errorf("in resp-text-code: %v", c.dec.Err())
		}
		// TODO: LONGENTRIES and MAXSIZE from METADATA
		switch code {
		case "CAPABILITY": // capability-data
			caps, err := readCapabilities(c.dec)
			if err != nil {
				return nil, fmt.Errorf("in capability-data: %v", err)
			}
			c.setCaps(caps)
		case "APPENDUID":
			var uidValidity, uid uint32
			if !c.dec.ExpectSP() || !c.dec.ExpectNumber(&uidValidity) || !c.dec.ExpectSP() || !c.dec.ExpectNumber(&uid) {
				return nil, fmt.Errorf("in resp-code-apnd: %v", c.dec.Err())
			}
			if cmd, ok := cmd.(*AppendCommand); ok {
				cmd.data.UID = uid
				cmd.data.UIDValidity = uidValidity
			}
		case "COPYUID":
			if !c.dec.ExpectSP() {
				return nil, c.dec.Err()
			}
			uidValidity, srcUIDs, dstUIDs, err := readRespCodeCopy(c.dec)
			if err != nil {
				return nil, fmt.Errorf("in resp-code-copy: %v", err)
			}
			if cmd, ok := cmd.(*CopyCommand); ok {
				cmd.data.UIDValidity = uidValidity
				cmd.data.SourceUIDs = srcUIDs
				cmd.data.DestUIDs = dstUIDs
			}
		default: // [SP 1*<any TEXT-CHAR except "]">]
			if c.dec.SP() {
				c.dec.DiscardUntilByte(']')
			}
		}
		if !c.dec.ExpectSpecial(']') || !c.dec.ExpectSP() {
			return nil, fmt.Errorf("in resp-text: %v", c.dec.Err())
		}
	}
	var text string
	if !c.dec.ExpectText(&text) {
		return nil, fmt.Errorf("in resp-text: %v", c.dec.Err())
	}

	var cmdErr error
	switch typ {
	case "OK":
		// nothing to do
	case "NO", "BAD":
		cmdErr = &imap.Error{
			Type: imap.StatusResponseType(typ),
			Code: imap.ResponseCode(code),
			Text: text,
		}
	default:
		return nil, fmt.Errorf("in resp-cond-state: expected OK, NO or BAD status condition, but got %v", typ)
	}

	c.completeCommand(cmd, cmdErr)

	var startTLS *startTLSCommand
	if cmd, ok := cmd.(*startTLSCommand); ok && cmdErr == nil {
		startTLS = cmd
	}

	if cmdErr == nil && code != "CAPABILITY" {
		switch cmd.(type) {
		case *startTLSCommand, *loginCommand, *authenticateCommand:
			c.setCaps(nil)
		}
	}

	return startTLS, nil
}

func (c *Client) readResponseData(typ string) error {
	// number SP ("EXISTS" / "RECENT" / "FETCH" / "EXPUNGE")
	var num uint32
	if typ[0] >= '0' && typ[0] <= '9' {
		v, err := strconv.ParseUint(typ, 10, 32)
		if err != nil {
			return err
		}

		num = uint32(v)
		if !c.dec.ExpectSP() || !c.dec.ExpectAtom(&typ) {
			return c.dec.Err()
		}
	}

	switch typ {
	case "OK", "PREAUTH", "NO", "BAD", "BYE": // resp-cond-state / resp-cond-bye / resp-cond-auth
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}

		var code string
		if c.dec.Special('[') { // resp-text-code
			if !c.dec.ExpectAtom(&code) {
				return fmt.Errorf("in resp-text-code: %v", c.dec.Err())
			}
			switch code {
			case "CAPABILITY": // capability-data
				caps, err := readCapabilities(c.dec)
				if err != nil {
					return fmt.Errorf("in capability-data: %v", err)
				}
				c.setCaps(caps)
			case "PERMANENTFLAGS":
				if !c.dec.ExpectSP() {
					return c.dec.Err()
				}
				flags, err := readFlagList(c.dec)
				if err != nil {
					return err
				}

				c.mutex.Lock()
				if c.state == StateSelected {
					c.mailbox = c.mailbox.copy()
					c.mailbox.PermanentFlags = flags
				}
				c.mutex.Unlock()

				if cmd := findPendingCmdByType[*SelectCommand](c); cmd != nil {
					cmd.data.PermanentFlags = flags
				} else if handler := c.options.unilateralDataHandler().Mailbox; handler != nil {
					handler(&UnilateralDataMailbox{PermanentFlags: flags})
				}
			case "UIDNEXT":
				if !c.dec.ExpectSP() {
					return c.dec.Err()
				}
				var uidNext uint32
				if !c.dec.ExpectNumber(&uidNext) {
					return c.dec.Err()
				}
				if cmd := findPendingCmdByType[*SelectCommand](c); cmd != nil {
					cmd.data.UIDNext = uidNext
				}
			case "UIDVALIDITY":
				if !c.dec.ExpectSP() {
					return c.dec.Err()
				}
				var uidValidity uint32
				if !c.dec.ExpectNumber(&uidValidity) {
					return c.dec.Err()
				}
				if cmd := findPendingCmdByType[*SelectCommand](c); cmd != nil {
					cmd.data.UIDValidity = uidValidity
				}
			case "COPYUID":
				if !c.dec.ExpectSP() {
					return c.dec.Err()
				}
				uidValidity, srcUIDs, dstUIDs, err := readRespCodeCopy(c.dec)
				if err != nil {
					return fmt.Errorf("in resp-code-copy: %v", err)
				}
				if cmd := findPendingCmdByType[*MoveCommand](c); cmd != nil {
					cmd.data.UIDValidity = uidValidity
					cmd.data.SourceUIDs = srcUIDs
					cmd.data.DestUIDs = dstUIDs
				}
			default: // [SP 1*<any TEXT-CHAR except "]">]
				if c.dec.SP() {
					c.dec.DiscardUntilByte(']')
				}
			}
			if !c.dec.ExpectSpecial(']') || !c.dec.ExpectSP() {
				return fmt.Errorf("in resp-text: %v", c.dec.Err())
			}
		}

		var text string
		if !c.dec.ExpectText(&text) {
			return fmt.Errorf("in resp-text: %v", c.dec.Err())
		}

		if code == "CLOSED" {
			c.setState(StateAuthenticated)
		}

		if !c.greetingRecv {
			switch typ {
			case "OK":
				c.setState(StateNotAuthenticated)
			case "PREAUTH":
				c.setState(StateAuthenticated)
			default:
				c.setState(StateLogout)
				c.greetingErr = &imap.Error{
					Type: imap.StatusResponseType(typ),
					Code: imap.ResponseCode(code),
					Text: text,
				}
			}
			c.greetingRecv = true
			close(c.greetingCh)
		}
	case "CAPABILITY":
		return c.handleCapability()
	case "ENABLED":
		return c.handleEnabled()
	case "NAMESPACE":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleNamespace()
	case "FLAGS":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleFlags()
	case "EXISTS":
		return c.handleExists(num)
	case "RECENT":
		// ignore
	case "LIST":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleList()
	case "STATUS":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleStatus()
	case "FETCH":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleFetch(num)
	case "EXPUNGE":
		return c.handleExpunge(num)
	case "SEARCH":
		return c.handleSearch()
	case "ESEARCH":
		return c.handleESearch()
	case "SORT":
		return c.handleSort()
	case "METADATA":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleMetadata()
	case "QUOTA":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleQuota()
	case "QUOTAROOT":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		return c.handleQuotaRoot()
	default:
		return fmt.Errorf("unsupported response type %q", typ)
	}

	return nil
}

// WaitGreeting waits for the server's initial greeting.
func (c *Client) WaitGreeting() error {
	<-c.greetingCh
	return c.greetingErr
}

// Noop sends a NOOP command.
func (c *Client) Noop() *Command {
	cmd := &Command{}
	c.beginCommand("NOOP", cmd).end()
	return cmd
}

// Logout sends a LOGOUT command.
//
// This command informs the server that the client is done with the connection.
func (c *Client) Logout() *Command {
	cmd := &logoutCommand{}
	c.beginCommand("LOGOUT", cmd).end()
	return &cmd.cmd
}

// Login sends a LOGIN command.
func (c *Client) Login(username, password string) *Command {
	cmd := &loginCommand{}
	enc := c.beginCommand("LOGIN", cmd)
	enc.SP().String(username).SP().String(password)
	enc.end()
	return &cmd.cmd
}

// Create sends a CREATE command.
func (c *Client) Create(mailbox string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("CREATE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Delete sends a DELETE command.
func (c *Client) Delete(mailbox string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("DELETE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Rename sends a RENAME command.
func (c *Client) Rename(mailbox, newName string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("RENAME", cmd)
	enc.SP().Mailbox(mailbox).SP().Mailbox(newName)
	enc.end()
	return cmd
}

// Subscribe sends a SUBSCRIBE command.
func (c *Client) Subscribe(mailbox string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("SUBSCRIBE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Subscribe sends an UNSUBSCRIBE command.
func (c *Client) Unsubscribe(mailbox string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("UNSUBSCRIBE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

func uidCmdName(name string, uid bool) string {
	if uid {
		return "UID " + name
	} else {
		return name
	}
}

type commandEncoder struct {
	*imapwire.Encoder
	client *Client
	cmd    *Command
}

// end ends an outgoing command.
//
// A CRLF is written, the encoder is flushed and its lock is released.
func (ce *commandEncoder) end() {
	if ce.Encoder != nil {
		ce.flush()
	}
	ce.client.setWriteTimeout(0)
	ce.client.encMutex.Unlock()
}

// flush sends an outgoing command, but keeps the encoder lock.
//
// A CRLF is written and the encoder is flushed. Callers must call
// commandEncoder.end to release the lock.
func (ce *commandEncoder) flush() {
	if err := ce.Encoder.CRLF(); err != nil {
		ce.cmd.err = err
	}
	ce.Encoder = nil
}

// Literal encodes a literal.
func (ce *commandEncoder) Literal(size int64) io.WriteCloser {
	var contReq *imapwire.ContinuationRequest
	if size > 4096 || !ce.client.Caps().Has(imap.CapLiteralMinus) {
		contReq = ce.client.registerContReq(ce.cmd)
	}
	ce.client.setWriteTimeout(literalWriteTimeout)
	return literalWriter{
		WriteCloser: ce.Encoder.Literal(size, contReq),
		client:      ce.client,
	}
}

type literalWriter struct {
	io.WriteCloser
	client *Client
}

func (lw literalWriter) Close() error {
	lw.client.setWriteTimeout(respWriteTimeout)
	return lw.WriteCloser.Close()
}

// continuationRequest is a pending continuation request.
type continuationRequest struct {
	*imapwire.ContinuationRequest
	cmd *Command
}

// UnilateralDataMailbox describes a mailbox status update.
//
// If a field is nil, it hasn't changed.
type UnilateralDataMailbox struct {
	NumMessages    *uint32
	Flags          []imap.Flag
	PermanentFlags []imap.Flag
}

// UnilateralDataHandler handles unilateral data.
//
// The handler will block the client while running. If the caller intends to
// perform slow operations, a buffered channel and a separate goroutine should
// be used.
//
// The handler will be invoked in an arbitrary goroutine.
//
// See Options.UnilateralDataHandler.
type UnilateralDataHandler struct {
	Expunge func(seqNum uint32)
	Mailbox func(data *UnilateralDataMailbox)
	Fetch   func(msg *FetchMessageData)
}

// command is an interface for IMAP commands.
//
// Commands are represented by the Command type, but can be extended by other
// types (e.g. CapabilityCommand).
type command interface {
	base() *Command
}

// Command is a basic IMAP command.
type Command struct {
	tag  string
	done chan error
	err  error
}

func (cmd *Command) base() *Command {
	return cmd
}

// Wait blocks until the command has completed.
func (cmd *Command) Wait() error {
	if cmd.err == nil {
		cmd.err = <-cmd.done
	}
	return cmd.err
}

type cmd = Command // type alias to avoid exporting anonymous struct fields

type loginCommand struct {
	cmd
}

// logoutCommand is a LOGOUT command.
type logoutCommand struct {
	cmd
}

func readFlagList(dec *imapwire.Decoder) ([]imap.Flag, error) {
	var flags []imap.Flag
	err := dec.ExpectList(func() error {
		flag, err := readFlag(dec)
		if err != nil {
			return err
		}
		flags = append(flags, imap.Flag(flag))
		return nil
	})
	return flags, err
}

func readFlag(dec *imapwire.Decoder) (string, error) {
	isSystem := dec.Special('\\')
	if isSystem && dec.Special('*') {
		return "\\*", nil // flag-perm
	}
	var name string
	if !dec.ExpectAtom(&name) {
		return "", fmt.Errorf("in flag: %v", dec.Err())
	}
	if isSystem {
		name = "\\" + name
	}
	return name, nil
}
