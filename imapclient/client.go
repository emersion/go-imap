package imapclient

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
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

func (c *Client) readResponse() error {
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
				// TODO: relax
				var s string
				if !c.dec.ExpectAtom(&s) {
					return c.dec.Err()
				}
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
