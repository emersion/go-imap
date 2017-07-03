// Package client provides an IMAP client.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/emersion/go-imap"
)

// errClosed is used when a connection is closed while waiting for a command
// response.
var errClosed = fmt.Errorf("imap: connection closed")

// Client is an IMAP client.
type Client struct {
	conn  *imap.Conn
	isTLS bool

	handles   imap.RespHandler
	handler   *imap.MultiRespHandler
	greeted   chan struct{}
	loggedOut chan struct{}

	// The cached server capabilities.
	caps map[string]bool
	// The caps map may be accessed in different goroutines. Protect access.
	capsLocker sync.Mutex

	// The current connection state.
	State imap.ConnState
	// The selected mailbox, if there is one.
	Mailbox *imap.MailboxStatus

	// A channel where info messages from the server will be sent.
	Infos chan *imap.StatusResp
	// A channel where warning messages from the server will be sent.
	Warnings chan *imap.StatusResp
	// A channel where error messages from the server will be sent.
	Errors chan *imap.StatusResp
	// A channel where bye messages from the server will be sent.
	Byes chan *imap.StatusResp
	// A channel where mailbox updates from the server will be sent.
	MailboxUpdates chan *imap.MailboxStatus
	// A channel where deleted message IDs will be sent.
	Expunges chan uint32
	// A channel where messages updates from the server will be sent.
	MessageUpdates chan *imap.Message

	// ErrorLog specifies an optional logger for errors accepting
	// connections and unexpected behavior from handlers.
	// If nil, logging goes to os.Stderr via the log package's
	// standard logger.
	ErrorLog imap.Logger

	// Timeout specifies a maximum amount of time to wait on a command.
	//
	// A Timeout of zero means no timeout. This is the default.
	Timeout time.Duration
}

func (c *Client) read(greeted <-chan struct{}) error {
	greetedClosed := false

	defer func() {
		// Ensure we close the greeted channel. New may be waiting on an indication
		// that we've seen the greeting.
		if !greetedClosed {
			close(c.greeted)
			greetedClosed = true
		}
		close(c.handles)
		close(c.loggedOut)
	}()

	first := true
	for {
		if c.State == imap.LogoutState {
			return nil
		}

		c.conn.Wait()

		if first {
			first = false
		} else {
			<-greeted
			if !greetedClosed {
				close(c.greeted)
				greetedClosed = true
			}
		}

		res, err := imap.ReadResp(c.conn.Reader)
		if err == io.EOF || c.State == imap.LogoutState {
			return nil
		}
		if err != nil {
			c.ErrorLog.Println("error reading response:", err)
			if imap.IsParseError(err) {
				continue
			} else {
				return err
			}
		}

		rh := &imap.RespHandle{
			Resp:    res,
			Accepts: make(chan bool),
		}
		c.handles <- rh
		if accepted := <-rh.Accepts; !accepted {
			c.ErrorLog.Println("response has not been handled:", res)
		}
	}
}

func (c *Client) execute(cmdr imap.Commander, res imap.RespHandlerFrom) (status *imap.StatusResp, err error) {
	cmd := cmdr.Command()
	cmd.Tag = generateTag()

	if c.Timeout > 0 {
		err = c.conn.SetDeadline(time.Now().Add(c.Timeout))
		if err != nil {
			return
		}
	} else {
		// It's possible the client had a timeout set from a previous command, but no
		// longer does. Ensure we respect that. The zero time means no deadline.
		err = c.conn.SetDeadline(time.Time{})
		if err != nil {
			return
		}
	}

	// Add handler before sending command, to be sure to get the response in time
	// (in tests, the response is sent right after our command is received, so
	// sometimes the response was received before the setup of this handler)
	statusHdlr := make(imap.RespHandler)
	c.handler.Add(statusHdlr)

	// Send the command to the server
	doneWrite := make(chan error, 1)
	go func() {
		doneWrite <- cmd.WriteTo(c.conn.Writer)
	}()

	// If a response handler is provided, start it
	var hdlr imap.RespHandler
	var doneHandle chan error
	if res != nil {
		hdlr = make(imap.RespHandler)
		doneHandle = make(chan error, 1)

		go func() {
			doneHandle <- res.HandleFrom(hdlr)
		}()
	}

	for {
		select {
		case <-c.loggedOut:
			// If the connection is closed (such as from an I/O error), ensure we
			// realize this and don't block waiting on a response that will never
			// come. loggedOut is a channel that closes when the reader goroutine
			// ends.
			err = errClosed
			return
		case err = <-doneWrite:
			// Error while sending the command
			if err != nil {
				c.handler.Del(statusHdlr)
			}
		case err = <-doneHandle:
			// Error while handling responses
			if err != nil {
				c.handler.Del(statusHdlr)
			}
		case h, more := <-statusHdlr:
			if !more {
				// statusHdlr has been closed, stop here
				return
			}

			// If the status tag matches the command tag, the response is completed
			if s, ok := h.Resp.(*imap.StatusResp); ok && s.Tag == cmd.Tag {
				h.Accept()
				status = s

				// Stop the response handler, if it's running
				if hdlr != nil {
					close(hdlr)
				}

				// Do not listen for responses anymore
				c.handler.Del(statusHdlr)
			} else if hdlr != nil {
				hdlr <- h
			} else {
				h.Reject()
			}
		}
	}
}

// Execute executes a generic command. cmdr is a value that can be converted to
// a raw command and res is a value that can handle responses. The function
// returns when the command has completed or failed, in this case err is nil. A
// non-nil err value indicates a network error.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (c *Client) Execute(cmdr imap.Commander, res imap.RespHandlerFrom) (status *imap.StatusResp, err error) {
	return c.execute(cmdr, res)
}

func (c *Client) handleContinuationReqs(continues chan<- bool) {
	hdlr := make(imap.RespHandler)
	c.handler.Add(hdlr)
	defer c.handler.Del(hdlr)

	defer close(continues)

	for h := range hdlr {
		if _, ok := h.Resp.(*imap.ContinuationResp); ok {
			h.Accept()
			continues <- true
		} else {
			h.Reject()
		}
	}
}

func (c *Client) gotStatusCaps(args []interface{}) {
	c.capsLocker.Lock()

	c.caps = make(map[string]bool)
	for _, cap := range args {
		if cap, ok := cap.(string); ok {
			c.caps[cap] = true
		}
	}

	c.capsLocker.Unlock()
}

// The server can send unilateral data. This function handles it.
func (c *Client) handleUnilateral() {
	hdlr := make(imap.RespHandler)
	c.handler.Add(hdlr)
	defer c.handler.Del(hdlr)

	greeted := make(chan struct{})

	// Make sure to start reading after we have set up the base handlers,
	// otherwise some messages will be lost.
	go c.read(greeted)

	first := true
	for h := range hdlr {
		switch res := h.Resp.(type) {
		case *imap.StatusResp:
			if res.Tag != "*" ||
				(res.Type != imap.StatusOk && res.Type != imap.StatusNo && res.Type != imap.StatusBad && res.Type != imap.StatusBye) ||
				(res.Code != "" && res.Code != imap.CodeAlert && res.Code != imap.CodeCapability) {
				h.Reject()
				break
			}
			h.Accept()

			if first {
				switch res.Type {
				case imap.StatusPreauth:
					c.State = imap.AuthenticatedState
				case imap.StatusBye:
					c.State = imap.LogoutState
				case imap.StatusOk:
					c.State = imap.NotAuthenticatedState
				default:
					c.ErrorLog.Println("invalid greeting:", res.Type)
					c.State = imap.LogoutState
				}

				if res.Code == imap.CodeCapability {
					c.gotStatusCaps(res.Arguments)
				}

				close(greeted)
				first = false
			}

			switch res.Type {
			case imap.StatusOk:
				if c.Infos != nil {
					c.Infos <- res
				}
			case imap.StatusNo:
				if c.Warnings != nil {
					c.Warnings <- res
				}
			case imap.StatusBad:
				if c.Errors != nil {
					c.Errors <- res
				}
			case imap.StatusBye:
				c.State = imap.LogoutState
				c.Mailbox = nil
				c.conn.Close()

				if c.Byes != nil {
					c.Byes <- res
				}
			}
		case *imap.Resp:
			if len(res.Fields) < 2 {
				h.Reject()
				break
			}

			// A CAPABILITY response
			if name, ok := res.Fields[0].(string); ok && name == imap.Capability {
				h.Accept()
				c.gotStatusCaps(res.Fields[1:len(res.Fields)])
				break
			}

			// An unilateral EXISTS, RECENT, EXPUNGE or FETCH response
			name, ok := res.Fields[1].(string)
			if !ok || (name != "EXISTS" && name != "RECENT" && name != "EXPUNGE" && name != "FETCH") {
				h.Reject()
				break
			}
			h.Accept()

			switch name {
			case "EXISTS":
				if c.Mailbox == nil {
					break
				}

				if messages, err := imap.ParseNumber(res.Fields[0]); err == nil {
					c.Mailbox.Messages = messages
					c.Mailbox.ItemsLocker.Lock()
					c.Mailbox.Items[imap.MailboxMessages] = nil
					c.Mailbox.ItemsLocker.Unlock()
				}

				if c.MailboxUpdates != nil {
					c.MailboxUpdates <- c.Mailbox
				}
			case "RECENT":
				if c.Mailbox == nil {
					break
				}

				if recent, err := imap.ParseNumber(res.Fields[0]); err == nil {
					c.Mailbox.Recent = recent
					c.Mailbox.ItemsLocker.Lock()
					c.Mailbox.Items[imap.MailboxRecent] = nil
					c.Mailbox.ItemsLocker.Unlock()
				}

				if c.MailboxUpdates != nil {
					c.MailboxUpdates <- c.Mailbox
				}
			case "EXPUNGE":
				seqNum, _ := imap.ParseNumber(res.Fields[0])

				if c.Expunges != nil {
					c.Expunges <- seqNum
				}
			case "FETCH":
				seqNum, _ := imap.ParseNumber(res.Fields[0])
				fields, _ := res.Fields[2].([]interface{})

				msg := &imap.Message{
					SeqNum: seqNum,
				}
				if err := msg.Parse(fields); err != nil {
					break
				}

				if c.MessageUpdates != nil {
					c.MessageUpdates <- msg
				}
			}
		default:
			h.Reject()
		}
	}
}

// Upgrade a connection, e.g. wrap an unencrypted connection with an encrypted
// tunnel.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (c *Client) Upgrade(upgrader imap.ConnUpgrader) error {
	return c.conn.Upgrade(upgrader)
}

// Writer returns the imap.Writer for this client's connection.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (c *Client) Writer() *imap.Writer {
	return c.conn.Writer
}

// IsTLS checks if this client's connection has TLS enabled.
func (c *Client) IsTLS() bool {
	return c.isTLS
}

// LoggedOut returns a channel which is closed when the connection to the server
// is closed.
func (c *Client) LoggedOut() <-chan struct{} {
	return c.loggedOut
}

// SetDebug defines an io.Writer to which all network activity will be logged.
// If nil is provided, network activity will not be logged.
func (c *Client) SetDebug(w io.Writer) {
	c.conn.SetDebug(w)
}

// New creates a new client from an existing connection.
func New(conn net.Conn) (c *Client, err error) {
	continues := make(chan bool)
	w := imap.NewClientWriter(nil, continues)
	r := imap.NewReader(nil)

	c = &Client{
		conn:      imap.NewConn(conn, r, w),
		handles:   make(imap.RespHandler),
		handler:   imap.NewMultiRespHandler(),
		greeted:   make(chan struct{}),
		loggedOut: make(chan struct{}),
		State:     imap.ConnectingState,
		ErrorLog:  log.New(os.Stderr, "imap/client: ", log.LstdFlags),
	}

	go c.handleContinuationReqs(continues)
	go c.handleUnilateral()
	go c.handler.HandleFrom(c.handles)

	<-c.greeted
	return
}

// Dial connects to an IMAP server using an unencrypted connection.
func Dial(addr string) (c *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}

	c, err = New(conn)
	return
}

// DialWithDialer connects to an IMAP server using an unencrypted connection
// using dialer.Dial.
//
// Among other uses, this allows us to apply a connection timeout.
func DialWithDialer(dialer *net.Dialer, address string) (c *Client, err error) {
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	// We don't return to the caller until we try to receive a greeting. As such,
	// there is no way to set the client's Timeout for that action. As a
	// workaround, if the dialer has a timeout set, use that for the connection's
	// deadline.
	if dialer.Timeout > 0 {
		err = conn.SetDeadline(time.Now().Add(dialer.Timeout))
		if err != nil {
			return
		}
	}

	c, err = New(conn)
	return
}

// DialTLS connects to an IMAP server using an encrypted connection.
func DialTLS(addr string, tlsConfig *tls.Config) (c *Client, err error) {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return
	}

	c, err = New(conn)
	c.isTLS = true
	return
}

// DialWithDialerTLS connects to an IMAP server using an encrypted connection
// using dialer.Dial.
//
// Among other uses, this allows us to apply a connection timeout.
func DialWithDialerTLS(dialer *net.Dialer, addr string,
	tlsConfig *tls.Config) (c *Client, err error) {
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return
	}

	// We don't return to the caller until we try to receive a greeting. As such,
	// there is no way to set the client's Timeout for that action. As a
	// workaround, if the dialer has a timeout set, use that for the connection's
	// deadline.
	if dialer.Timeout > 0 {
		err = conn.SetDeadline(time.Now().Add(dialer.Timeout))
		if err != nil {
			return
		}
	}

	c, err = New(conn)
	c.isTLS = true
	return
}
