package imapclient

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/emersion/go-imap/v2/internal/imapwire"

	"os"
)

var debug = true

// Client is an IMAP client.
//
// IMAP commands are exposed as methods. These methods will block until the
// command has been sent to the server, but won't block until the server sends
// a response. They return a command struct which can be used to wait for the
// server response, see e.g. Command.
type Client struct {
	conn     net.Conn
	dec      *imapwire.Decoder
	enc      *imapwire.Encoder
	encMutex sync.Mutex

	mutex       sync.Mutex
	cmdTag      uint64
	pendingCmds []command
	contReqs    []chan<- struct{}
}

// New creates a new IMAP client.
//
// This function doesn't perform I/O.
func New(conn net.Conn) *Client {
	var (
		r io.Reader = conn
		w io.Writer = conn
	)
	if debug {
		r = io.TeeReader(r, os.Stderr)
		w = io.MultiWriter(w, os.Stderr)
	}

	client := &Client{
		conn: conn,
		dec:  imapwire.NewDecoder(bufio.NewReader(r)),
		enc:  imapwire.NewEncoder(bufio.NewWriter(w)),
	}
	go client.read()
	return client
}

// DialTLS connects to an IMAP server with implicit TLS.
func DialTLS(address string) (*Client, error) {
	conn, err := tls.Dial("tcp", address, nil)
	if err != nil {
		return nil, err
	}
	return New(conn), nil
}

// Close immediately closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// beginCommand starts sending a command to the server.
//
// The command name and a space are written.
//
// The caller must call endCommand.
func (c *Client) beginCommand(name string) *Command {
	c.encMutex.Lock() // unlocked by endCommand

	c.mutex.Lock()
	c.cmdTag++
	tag := fmt.Sprintf("T%v", c.cmdTag)
	c.mutex.Unlock()

	cmd := &Command{
		tag:  tag,
		done: make(chan error, 1),
		enc:  c.enc,
	}
	cmd.enc.Atom(tag).SP().Atom(name)
	return cmd
}

// endCommand ends an outgoing command.
//
// A CRLF is written.
//
// The command is registered as a pending command until the server sends a
// completion result response.
func (c *Client) endCommand(cmd command) {
	baseCmd := cmd.base()

	c.mutex.Lock()
	c.pendingCmds = append(c.pendingCmds, cmd)
	c.mutex.Unlock()

	if err := baseCmd.enc.CRLF(); err != nil {
		baseCmd.err = err
	}
	baseCmd.enc = nil
	c.encMutex.Unlock()
}

func (c *Client) deletePendingCmdByTag(tag string) *Command {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for i, cmd := range c.pendingCmds {
		cmdBase := cmd.base()
		if cmdBase.tag == tag {
			c.pendingCmds = append(c.pendingCmds[:i], c.pendingCmds[i+1:]...)
			return cmdBase
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

func (c *Client) encodeLiteral(size int64) io.WriteCloser {
	ch := make(chan struct{})

	c.mutex.Lock()
	c.contReqs = append(c.contReqs, ch)
	c.mutex.Unlock()

	return c.enc.Literal(size, ch)
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
		token string
		err   error
	)
	if tag != "" {
		token = "response-tagged"
		err = c.readResponseTagged(tag, typ)
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

	return nil
}

func (c *Client) readContinueReq() error {
	var text string
	if !c.dec.ExpectSP() || !c.dec.ExpectText(&text) || !c.dec.ExpectCRLF() {
		return c.dec.Err()
	}

	var ch chan<- struct{}
	c.mutex.Lock()
	if len(c.contReqs) > 0 {
		ch = c.contReqs[0]
		c.contReqs = append(c.contReqs[:0], c.contReqs[1:]...)
	}
	c.mutex.Unlock()

	if ch == nil {
		return fmt.Errorf("received unmatched continuation request")
	}

	close(ch)
	return nil
}

func (c *Client) readResponseTagged(tag, typ string) error {
	if !c.dec.ExpectSP() {
		return c.dec.Err()
	}
	if c.dec.Special('[') { // resp-text-code
		var code string
		if !c.dec.ExpectAtom(&code) {
			return fmt.Errorf("in resp-text-code: %v", c.dec.Err())
		}
		switch code {
		case "CAPABILITY": // capability-data
			if _, err := readCapabilities(c.dec); err != nil {
				return err
			}
		default: // [SP 1*<any TEXT-CHAR except "]">]
			if c.dec.SP() {
				c.dec.Skip(']')
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

	var cmdErr error
	switch typ {
	case "OK":
		// nothing to do
	case "NO", "BAD":
		// TODO: define a type for IMAP errors
		cmdErr = fmt.Errorf("%v %v", typ, text)
	default:
		return fmt.Errorf("in resp-cond-state: expected OK, NO or BAD status condition, but got %v", typ)
	}

	cmd := c.deletePendingCmdByTag(tag)
	if cmd == nil {
		return fmt.Errorf("received tagged response with unknown tag %q", tag)
	}
	cmd.done <- cmdErr
	close(cmd.done)
	return nil
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
		// TODO: handle flags
		_, err := readFlagList(c.dec)
		return err
	case "EXISTS", "RECENT":
		_ = num // TODO: handle
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

// Noop sends a NOOP command.
func (c *Client) Noop() *Command {
	cmd := c.beginCommand("NOOP")
	c.endCommand(cmd)
	return cmd
}

// Logout sends a LOGOUT command.
//
// This command informs the server that the client is done with the connection.
func (c *Client) Logout() *LogoutCommand {
	cmd := &LogoutCommand{c.beginCommand("LOGOUT"), c}
	c.endCommand(cmd)
	return cmd
}

// Capability sends a CAPABILITY command.
func (c *Client) Capability() *CapabilityCommand {
	cmd := &CapabilityCommand{cmd: c.beginCommand("CAPABILITY")}
	c.endCommand(cmd)
	return cmd
}

// Login sends a LOGIN command.
func (c *Client) Login(username, password string) *Command {
	cmd := c.beginCommand("LOGIN")
	cmd.enc.SP().String(username).SP().String(password)
	c.endCommand(cmd)
	return cmd
}

// Append sends an APPEND command.
//
// The caller must call AppendCommand.Close.
func (c *Client) Append(mailbox string, size int64) *AppendCommand {
	// TODO: flag parenthesized list, date/time string
	cmd := c.beginCommand("APPEND")
	cmd.enc.SP().Mailbox(mailbox).SP()
	wc := c.encodeLiteral(size)
	return &AppendCommand{
		cmd:    cmd,
		client: c,
		wc:     wc,
	}
}

// Select sends a SELECT command.
func (c *Client) Select(mailbox string) *Command {
	cmd := c.beginCommand("SELECT")
	cmd.enc.SP().Mailbox(mailbox)
	c.endCommand(cmd)
	return cmd
}

// Fetch sends a FETCH command.
//
// The caller must fully consume the FetchCommand. A simple way to do so is to
// defer a call to FetchCommand.Close.
func (c *Client) Fetch(seqNum uint32) *FetchCommand {
	// TODO: sequence set, message data item names or macro
	cmd := &FetchCommand{
		cmd:  c.beginCommand("FETCH"),
		msgs: make(chan *FetchMessageData, 128),
	}
	cmd.enc.SP().Number(seqNum).SP().Special('(').Atom("BODY[]").Special(')')
	c.endCommand(cmd)
	return cmd
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
	enc  *imapwire.Encoder
	err  error
}

func (cmd *Command) base() *Command {
	return cmd
}

// Wait blocks until the command has completed.
func (cmd *Command) Wait() error {
	if cmd.enc != nil {
		panic("command waited before being closed")
	}
	if cmd.err == nil {
		cmd.err = <-cmd.done
	}
	return cmd.err
}

type cmd = Command // type alias to avoid exporting anonymous struct fields

// CapabilityCommand is a CAPABILITY command.
type CapabilityCommand struct {
	*cmd
	caps map[string]struct{}
}

func (cmd *CapabilityCommand) Wait() (map[string]struct{}, error) {
	err := cmd.cmd.Wait()
	return cmd.caps, err
}

// LogoutCommand is a LOGOUT command.
type LogoutCommand struct {
	*cmd
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
	*cmd
	client *Client
	wc     io.WriteCloser
}

func (cmd *AppendCommand) Write(b []byte) (int, error) {
	return cmd.wc.Write(b)
}

func (cmd *AppendCommand) Close() error {
	err := cmd.wc.Close()
	if cmd.client != nil {
		cmd.client.endCommand(cmd)
		cmd.client = nil
	}
	return err
}

func (cmd *AppendCommand) Wait() error {
	return cmd.cmd.Wait()
}

// FetchCommand is a FETCH command.
type FetchCommand struct {
	*cmd
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
