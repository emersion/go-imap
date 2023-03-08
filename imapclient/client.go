package imapclient

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Options contains options for Client.
type Options struct {
	// Raw ingress and egress data will be written to this writer, if any
	DebugWriter io.Writer
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

// Client is an IMAP client.
//
// IMAP commands are exposed as methods. These methods will block until the
// command has been sent to the server, but won't block until the server sends
// a response. They return a command struct which can be used to wait for the
// server response, see e.g. Command.
type Client struct {
	conn     net.Conn
	options  Options
	br       *bufio.Reader
	bw       *bufio.Writer
	dec      *imapwire.Decoder
	encMutex sync.Mutex

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
		conn:    conn,
		options: *options,
		br:      br,
		bw:      bw,
		dec:     imapwire.NewDecoder(br),
	}
	go client.read()
	return client
}

// DialTLS connects to an IMAP server with implicit TLS.
func DialTLS(address string, options *Options) (*Client, error) {
	conn, err := tls.Dial("tcp", address, nil)
	if err != nil {
		return nil, err
	}
	return New(conn, options), nil
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

func (c *Client) registerContReq(cmd command) chan error {
	ch := make(chan error)

	c.mutex.Lock()
	c.contReqs = append(c.contReqs, continuationRequest{
		ch:  ch,
		cmd: cmd.base(),
	})
	c.mutex.Unlock()

	return ch
}

func (c *Client) unregisterContReq(ch chan error) {
	c.mutex.Lock()
	for i := range c.contReqs {
		if c.contReqs[i].ch == ch {
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
		c.mutex.Lock()
		pendingCmds := c.pendingCmds
		c.pendingCmds = nil
		c.mutex.Unlock()

		for _, cmd := range pendingCmds {
			cmd.base().done <- io.ErrUnexpectedEOF
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
	if !c.dec.ExpectSP() || !c.dec.ExpectText(&text) || !c.dec.ExpectCRLF() {
		return c.dec.Err()
	}

	var ch chan<- error
	c.mutex.Lock()
	if len(c.contReqs) > 0 {
		ch = c.contReqs[0].ch
		c.contReqs = append(c.contReqs[:0], c.contReqs[1:]...)
	}
	c.mutex.Unlock()

	if ch == nil {
		return fmt.Errorf("received unmatched continuation request")
	}

	ch <- nil
	close(ch)
	return nil
}

func (c *Client) readResponseTagged(tag, typ string) (*startTLSCommand, error) {
	if !c.dec.ExpectSP() {
		return nil, c.dec.Err()
	}
	if c.dec.Special('[') { // resp-text-code
		var code string
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
		// TODO: define a type for IMAP errors
		cmdErr = fmt.Errorf("%v %v", typ, text)
	default:
		return nil, fmt.Errorf("in resp-cond-state: expected OK, NO or BAD status condition, but got %v", typ)
	}

	cmd := c.deletePendingCmdByTag(tag)
	if cmd == nil {
		return nil, fmt.Errorf("received tagged response with unknown tag %q", tag)
	}

	done := cmd.base().done
	done <- cmdErr
	close(done)

	// Ensure the command is not blocked waiting on continuation requests
	c.mutex.Lock()
	var filtered []continuationRequest
	for _, contReq := range c.contReqs {
		if contReq.cmd != cmd.base() {
			filtered = append(filtered, contReq)
		} else {
			if cmdErr != nil {
				contReq.ch <- cmdErr
			}
			close(contReq.ch)
		}
	}
	c.contReqs = filtered
	c.mutex.Unlock()

	var startTLS *startTLSCommand
	if cmd, ok := cmd.(*startTLSCommand); ok && cmdErr == nil {
		startTLS = cmd
	}

	return startTLS, nil
}

func (c *Client) readResponseData(typ string) error {
	// number SP "EXISTS" / number SP "RECENT"
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
	case "OK", "NO", "BAD": // resp-cond-state
		// TODO
		var text string
		if !c.dec.ExpectText(&text) {
			return fmt.Errorf("in resp-text: %v", c.dec.Err())
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
		}
	case "RECENT":
		// ignore
	case "FETCH":
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		cmd := findPendingCmdByType[*FetchCommand](c)
		if err := readMsgAtt(c.dec, cmd); err != nil {
			return fmt.Errorf("in msg-att: %v", err)
		}
	default:
		return fmt.Errorf("unsupported response type %q", typ)
	}
	return nil
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

// Capability sends a CAPABILITY command.
func (c *Client) Capability() *CapabilityCommand {
	cmd := &CapabilityCommand{}
	c.beginCommand("CAPABILITY", cmd).end()
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

// StartTLS sends a STARTTLS command.
//
// Unlike other commands, this method blocks until the command completes.
func (c *Client) StartTLS(config *tls.Config) error {
	upgradeDone := make(chan struct{})
	cmd := &startTLSCommand{
		tlsConfig:   config,
		upgradeDone: upgradeDone,
	}
	c.beginCommand("STARTTLS", cmd).end()

	// Once a client issues a STARTTLS command, it MUST NOT issue further
	// commands until a server response is seen and the TLS negotiation is
	// complete
	// TODO: race if another goroutine sends a command between endCommand and
	// encMutex.Lock
	c.encMutex.Lock()
	defer c.encMutex.Unlock()

	if err := cmd.Wait(); err != nil {
		return err
	}

	// The decoder goroutine will invoke Client.upgradeStartTLS
	<-upgradeDone
	return nil
}

func (c *Client) upgradeStartTLS(tlsConfig *tls.Config) {
	// Drain buffered data from our bufio.Reader
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, c.br, int64(c.br.Buffered())); err != nil {
		panic(err) // unreachable
	}

	var cleartextConn net.Conn
	if buf.Len() > 0 {
		r := io.MultiReader(&buf, c.conn)
		cleartextConn = startTLSConn{c.conn, r}
	} else {
		cleartextConn = c.conn
	}

	tlsConn := tls.Client(cleartextConn, tlsConfig)
	rw := c.options.wrapReadWriter(tlsConn)

	c.br.Reset(rw)
	// Unfortunately we can't re-use the bufio.Writer here, it races with
	// Client.StartTLS
	c.bw = bufio.NewWriter(rw)
}

// Append sends an APPEND command.
//
// The caller must call AppendCommand.Close.
func (c *Client) Append(mailbox string, size int64) *AppendCommand {
	// TODO: flag parenthesized list, date/time string
	cmd := &AppendCommand{}
	cmd.enc = c.beginCommand("APPEND", cmd)
	cmd.enc.SP().Mailbox(mailbox).SP()
	cmd.wc = cmd.enc.Literal(size)
	return cmd
}

// Select sends a SELECT command.
func (c *Client) Select(mailbox string) *SelectCommand {
	cmd := &SelectCommand{}
	enc := c.beginCommand("SELECT", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Fetch sends a FETCH command.
//
// The caller must fully consume the FetchCommand. A simple way to do so is to
// defer a call to FetchCommand.Close.
func (c *Client) Fetch(seqNum uint32) *FetchCommand {
	// TODO: sequence set, message data item names or macro
	cmd := &FetchCommand{
		msgs: make(chan *FetchMessageData, 128),
	}
	enc := c.beginCommand("FETCH", cmd)
	enc.SP().Number(seqNum).SP().Special('(').Atom("BODY[]").Special(')')
	enc.end()
	return cmd
}

// Idle sends an IDLE command.
//
// Unlike other commands, this method blocks until the server acknowledges it.
// On success, the IDLE command is running and other commands cannot be sent.
// The caller must invoke IdleCommand.Close to stop IDLE and unblock the
// client.
func (c *Client) Idle() (*IdleCommand, error) {
	cmd := &IdleCommand{client: c}
	contReq := c.registerContReq(cmd)
	c.beginCommand("IDLE", cmd).end()

	// encMutex is unlocked by IdleCommand.Close
	// TODO: race if another goroutine sends a command between endCommand and
	// encMutex.Lock
	c.encMutex.Lock()

	select {
	case <-contReq:
		return cmd, nil
	case err := <-cmd.done:
		c.unregisterContReq(contReq)
		c.encMutex.Unlock()
		return nil, err
	}
}

type commandEncoder struct {
	*imapwire.Encoder
	client *Client
	cmd    *Command
}

// end ends an outgoing command.
//
// A CRLF is written and the encoder is flushed.
func (ce *commandEncoder) end() {
	if err := ce.Encoder.CRLF(); err != nil {
		ce.cmd.err = err
	}
	ce.client.encMutex.Unlock()
	ce.Encoder = nil
}

// Literal encodes a literal.
func (ce *commandEncoder) Literal(size int64) io.WriteCloser {
	contReq := ce.client.registerContReq(ce.cmd)
	return ce.Encoder.Literal(size, contReq)
}

// continuationRequest is a pending continuation request.
type continuationRequest struct {
	ch  chan error
	cmd *Command
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

// CapabilityCommand is a CAPABILITY command.
type CapabilityCommand struct {
	cmd
	caps map[string]struct{}
}

func (cmd *CapabilityCommand) Wait() (map[string]struct{}, error) {
	err := cmd.cmd.Wait()
	return cmd.caps, err
}

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

// AppendCommand is an APPEND command.
//
// Callers must write the message contents, then call Close.
type AppendCommand struct {
	cmd
	enc *commandEncoder
	wc  io.WriteCloser
}

func (cmd *AppendCommand) Write(b []byte) (int, error) {
	return cmd.wc.Write(b)
}

func (cmd *AppendCommand) Close() error {
	err := cmd.wc.Close()
	if cmd.enc != nil {
		cmd.enc.end()
		cmd.enc = nil
	}
	return err
}

func (cmd *AppendCommand) Wait() error {
	return cmd.cmd.Wait()
}

// SelectCommand is a SELECT command.
type SelectCommand struct {
	cmd
	data SelectData
}

func (cmd *SelectCommand) Wait() (*SelectData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// SelectData is the data returned by a SELECT command.
type SelectData struct {
	// Flags defined for this mailbox
	Flags []string
	// Number of messages in this mailbox (aka. "EXISTS")
	NumMessages uint32

	// TODO: LIST, PERMANENTFLAGS, UIDNEXT, UIDVALIDITY
}

type startTLSCommand struct {
	cmd
	tlsConfig   *tls.Config
	upgradeDone chan<- struct{}
}

// FetchCommand is a FETCH command.
type FetchCommand struct {
	cmd
	msgs chan *FetchMessageData
	prev *FetchMessageData
}

// Next advances to the next message.
//
// On success, the message and a nil error is returned. If there are no more
// messages, io.EOF is returned. Otherwise the error is returned.
func (cmd *FetchCommand) Next() (*FetchMessageData, error) {
	if cmd.prev != nil {
		cmd.prev.discard()
	}

	select {
	case msg := <-cmd.msgs:
		cmd.prev = msg
		return msg, nil
	case err := <-cmd.done:
		if err == nil {
			return nil, io.EOF
		}
		return nil, err
	}
}

// Close releases the command.
//
// Calling Close unblocks the IMAP client decoder and lets it read the next
// responses. Next will always return an error after Close.
func (cmd *FetchCommand) Close() error {
	for {
		_, err := cmd.Next()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
	}
}

// FetchMessageData contains a message's FETCH data.
type FetchMessageData struct {
	items chan *FetchItemData
	prev  *FetchItemData
}

// Next advances to the next data item for this message.
//
// If there is one or more data items left, the next item is returned.
// Otherwise nil is returned.
func (data *FetchMessageData) Next() *FetchItemData {
	if data.prev != nil {
		data.prev.discard()
	}

	item := <-data.items
	data.prev = item
	return item
}

func (data *FetchMessageData) discard() {
	for {
		if item := data.Next(); item == nil {
			break
		}
	}
}

// FetchItemData contains a message's FETCH item data.
type FetchItemData struct {
	Name    string
	Literal LiteralReader
}

func (item *FetchItemData) discard() {
	io.Copy(io.Discard, item.Literal)
}

// LiteralReader is a reader for IMAP literals.
type LiteralReader interface {
	io.Reader
	Size() int64
}

type startTLSConn struct {
	net.Conn
	r io.Reader
}

func (conn startTLSConn) Read(b []byte) (int, error) {
	return conn.r.Read(b)
}

// IdleCommand is an IDLE command.
//
// Initially, the IDLE command is running. The server may send unilateral
// data. The client cannot send any command while IDLE is running.
//
// Close must be called to stop the IDLE command.
type IdleCommand struct {
	cmd
	client *Client
}

// Close stops the IDLE command.
//
// This method blocks until the command to stop IDLE is written, but doesn't
// wait for the server to respond. Callers can use Wait for this purpose.
func (cmd *IdleCommand) Close() error {
	if cmd.err != nil {
		return cmd.err
	}
	if cmd.client == nil {
		return fmt.Errorf("imapclient: IDLE command closed twice")
	}
	_, err := cmd.client.bw.WriteString("DONE\r\n")
	if err == nil {
		err = cmd.client.bw.Flush()
	}
	cmd.client.encMutex.Unlock()
	cmd.client = nil
	return err
}

// Wait blocks until the IDLE command has completed.
//
// Wait can only be called after Close.
func (cmd *IdleCommand) Wait() error {
	if cmd.client != nil {
		return fmt.Errorf("imapclient: IdleCommand.Close must be called before Wait")
	}
	return cmd.cmd.Wait()
}
