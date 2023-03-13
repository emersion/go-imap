package imapclient

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"strconv"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const dateTimeLayout = "_2-Jan-2006 15:04:05 -0700"

// Options contains options for Client.
type Options struct {
	// Raw ingress and egress data will be written to this writer, if any
	DebugWriter io.Writer
	// Unilateral data handler
	UnilateralDataFunc UnilateralDataFunc
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

// Client is an IMAP client.
//
// IMAP commands are exposed as methods. These methods will block until the
// command has been sent to the server, but won't block until the server sends
// a response. They return a command struct which can be used to wait for the
// server response.
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

	mutex       sync.Mutex
	cmdTag      uint64
	pendingCmds []command
	contReqs    []continuationRequest
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

// Close immediately closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
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

func findPendingCmdByType[T interface{}](c *Client) T {
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
	case *ListCommand:
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
	defer func() {
		if v := recover(); v != nil {
			// TODO: handle error
			log.Println(v)
		}

		c.Close()

		c.mutex.Lock()
		pendingCmds := c.pendingCmds
		c.pendingCmds = nil
		c.mutex.Unlock()

		for _, cmd := range pendingCmds {
			c.completeCommand(cmd, io.ErrUnexpectedEOF)
		}
	}()

	for {
		if c.dec.EOF() {
			break
		}
		if err := c.readResponse(); err != nil {
			// TODO: handle error
			log.Println(err)
			break
		}
		if c.greetingErr != nil {
			break
		}
	}
}

func (c *Client) readResponse() error {
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
	if !c.dec.ExpectSP() {
		return nil, c.dec.Err()
	}
	var code string
	if c.dec.Special('[') { // resp-text-code
		if !c.dec.ExpectAtom(&code) {
			return nil, fmt.Errorf("in resp-text-code: %v", c.dec.Err())
		}
		switch code {
		case "CAPABILITY": // capability-data
			if _, err := readCapabilities(c.dec); err != nil {
				return nil, err
			}
		default: // [SP 1*<any TEXT-CHAR except "]">]
			if c.dec.SP() {
				c.dec.Skip(']')
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

	cmd := c.deletePendingCmdByTag(tag)
	if cmd == nil {
		return nil, fmt.Errorf("received tagged response with unknown tag %q", tag)
	}

	c.completeCommand(cmd, cmdErr)

	var startTLS *startTLSCommand
	if cmd, ok := cmd.(*startTLSCommand); ok && cmdErr == nil {
		startTLS = cmd
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

	var unilateralData UnilateralData
	switch typ {
	case "OK", "NO", "BAD", "BYE": // resp-cond-state / resp-cond-bye
		// TODO: decode response code
		var text string
		if !c.dec.ExpectText(&text) {
			return fmt.Errorf("in resp-text: %v", c.dec.Err())
		}

		if !c.greetingRecv {
			if typ != "OK" {
				c.greetingErr = &imap.Error{
					Type: imap.StatusResponseType(typ),
					Text: text,
				}
			}
			c.greetingRecv = true
			close(c.greetingCh)
		}
	case "CAPABILITY": // capability-data
		caps, err := readCapabilities(c.dec)
		if err != nil {
			return err
		}
		if cmd := findPendingCmdByType[*CapabilityCommand](c); cmd != nil {
			cmd.caps = caps
		}
	case "FLAGS":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		flags, err := readFlagList(c.dec)
		if err != nil {
			return err
		}
		cmd := findPendingCmdByType[*SelectCommand](c)
		if cmd != nil {
			cmd.data.Flags = flags
		}
	case "EXISTS":
		cmd := findPendingCmdByType[*SelectCommand](c)
		if cmd != nil {
			cmd.data.NumMessages = num
		} else {
			unilateralData = &UnilateralDataMailbox{NumMessages: &num}
		}
	case "RECENT":
		// ignore
	case "LIST":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		data, err := readList(c.dec)
		if err != nil {
			return fmt.Errorf("in LIST: %v", err)
		}
		if cmd := findPendingCmdByType[*ListCommand](c); cmd != nil {
			cmd.mailboxes <- data
		}
	case "STATUS":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		cmd := findPendingCmdByType[*StatusCommand](c)
		if err := readStatus(c.dec, cmd); err != nil {
			return fmt.Errorf("in status: %v", err)
		}
	case "FETCH":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		cmd := findPendingCmdByType[*FetchCommand](c)
		if err := readMsgAtt(c.dec, num, cmd, &c.options); err != nil {
			return fmt.Errorf("in msg-att: %v", err)
		}
	case "EXPUNGE":
		cmd := findPendingCmdByType[*ExpungeCommand](c)
		if cmd != nil {
			cmd.seqNums <- num
		} else {
			unilateralData = &UnilateralDataExpunge{SeqNum: num}
		}
	case "SEARCH":
		// TODO: handle ESEARCH
		cmd := findPendingCmdByType[*SearchCommand](c)
		for c.dec.SP() {
			num, ok := c.dec.ExpectNumber()
			if !ok {
				return c.dec.Err()
			}
			if cmd != nil {
				cmd.data.All.AddNum(num)
			}
		}
	default:
		return fmt.Errorf("unsupported response type %q", typ)
	}

	if unilateralData != nil && c.options.UnilateralDataFunc != nil {
		c.options.UnilateralDataFunc(unilateralData)
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
func (c *Client) Logout() *LogoutCommand {
	cmd := &LogoutCommand{closer: c}
	c.beginCommand("LOGOUT", cmd).end()
	return cmd
}

// Login sends a LOGIN command.
func (c *Client) Login(username, password string) *Command {
	cmd := &Command{}
	enc := c.beginCommand("LOGIN", cmd)
	enc.SP().String(username).SP().String(password)
	enc.end()
	return cmd
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
	contReq := ce.client.registerContReq(ce.cmd)
	return ce.Encoder.Literal(size, contReq)
}

// continuationRequest is a pending continuation request.
type continuationRequest struct {
	*imapwire.ContinuationRequest
	cmd *Command
}

// UnilateralData holds unilateral data.
//
// Unilateral data is data that the client didn't explicitly request.
type UnilateralData interface {
	unilateralData()
}

var (
	_ UnilateralData = (*UnilateralDataMailbox)(nil)
	_ UnilateralData = (*UnilateralDataExpunge)(nil)
)

// UnilateralDataMailbox describes a mailbox status update.
//
// If a field is nil, it hasn't changed.
type UnilateralDataMailbox struct {
	NumMessages *uint32
}

func (*UnilateralDataMailbox) unilateralData() {}

// UnilateralDataExpunge indicates that a message has been deleted.
type UnilateralDataExpunge struct {
	SeqNum uint32
}

func (*UnilateralDataExpunge) unilateralData() {}

// UnilateralDataFunc handles unilateral data.
//
// The handler will block the client while running. If the caller intends to
// perform slow operations, a buffered channel and a separate goroutine should
// be used.
//
// The handler will be invoked in an arbitrary goroutine.
//
// See Options.UnilateralDataFunc.
type UnilateralDataFunc func(data UnilateralData)

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

// LogoutCommand is a LOGOUT command.
type LogoutCommand struct {
	cmd
	closer io.Closer
}

func (cmd *LogoutCommand) Wait() error {
	if err := cmd.cmd.Wait(); err != nil {
		return err
	}
	return cmd.closer.Close()
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
	var name string
	if !dec.ExpectAtom(&name) {
		return "", fmt.Errorf("in flag: %v", dec.Err())
	}
	if isSystem {
		name = "\\" + name
	}
	return name, nil
}
