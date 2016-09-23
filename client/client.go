// An IMAP client.
package client

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/emersion/go-imap"
)

// An IMAP client.
type Client struct {
	conn  *imap.Conn
	isTLS bool

	handlers       []imap.RespHandler
	handlersLocker sync.Locker
	greeted        chan struct{}
	loggedOut      chan struct{}

	// The server capabilities.
	Caps map[string]bool
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

	// TODO: support unilateral message updates
	// A channel where messages updates from the server will be sent.
	//MessageUpdates chan *imap.Message

	// ErrorLog specifies an optional logger for errors accepting
	// connections and unexpected behavior from handlers.
	// If nil, logging goes to os.Stderr via the log package's
	// standard logger.
	ErrorLog *log.Logger
}

func (c *Client) read(greeted chan struct{}) error {
	r := c.conn.Reader

	defer (func() {
		c.handlersLocker.Lock()
		defer c.handlersLocker.Unlock()

		for _, hdlr := range c.handlers {
			close(hdlr)
		}
		c.handlers = nil

		close(c.loggedOut)
	})()

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
			if c.greeted != nil {
				close(c.greeted)
				c.greeted = nil
			}
		}

		res, err := imap.ReadResp(r)
		if err == io.EOF || c.State == imap.LogoutState {
			return nil
		}
		if err != nil {
			c.ErrorLog.Println("error reading response: ", err)
			if imap.IsParseError(err) {
				continue
			} else {
				return err
			}
		}

		c.handlersLocker.Lock()

		var accepted bool
		for _, hdlr := range c.handlers {
			if hdlr == nil {
				continue
			}

			h := &imap.RespHandling{
				Resp:    res,
				Accepts: make(chan bool),
			}

			hdlr <- h

			accepted = <-h.Accepts
			if accepted {
				break
			}
		}

		c.handlersLocker.Unlock()

		if !accepted {
			c.ErrorLog.Println("response has not been handled: ", res)
		}
	}

	return nil
}

func (c *Client) addHandler(hdlr imap.RespHandler) {
	c.handlersLocker.Lock()
	defer c.handlersLocker.Unlock()

	c.handlers = append(c.handlers, hdlr)
}

func (c *Client) removeHandler(hdlr imap.RespHandler) {
	c.handlersLocker.Lock()
	defer c.handlersLocker.Unlock()

	for i, h := range c.handlers {
		if h == hdlr {
			close(hdlr)
			c.handlers = append(c.handlers[:i], c.handlers[i+1:]...)
		}
	}
}

func (c *Client) execute(cmdr imap.Commander, res imap.RespHandlerFrom) (status *imap.StatusResp, err error) {
	cmd := cmdr.Command()
	cmd.Tag = generateTag()

	// Add handler before sending command, to be sure to get the response in time
	// (in tests, the response is sent right after our command is received, so
	// sometimes the response was received before the setup of this handler)
	statusHdlr := make(imap.RespHandler)
	c.addHandler(statusHdlr)
	defer c.removeHandler(statusHdlr)

	if err = cmd.WriteTo(c.conn.Writer); err != nil {
		return
	}

	var hdlr imap.RespHandler
	var done chan error
	if res != nil {
		hdlr = make(imap.RespHandler)
		done = make(chan error, 1)

		go (func() {
			done <- res.HandleFrom(hdlr)

			if hdlr != nil {
				close(hdlr)
				hdlr = nil
			}
		})()
	}

	for h := range statusHdlr {
		if s, ok := h.Resp.(*imap.StatusResp); ok && s.Tag == cmd.Tag {
			h.Accept()
			status = s

			if hdlr != nil {
				close(hdlr)
				hdlr = nil
			}
			break
		} else if hdlr != nil {
			hdlr <- h
		} else {
			h.Reject()
		}
	}

	if done != nil {
		err = <-done
	}
	return
}

// Execute a generic command. cmdr is a value that can be converted to a raw
// command and res is a value that can handle responses. The function returns
// when the command has completed or failed, in this case err is nil. A non-nil
// err value indicates a network error.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (c *Client) Execute(cmdr imap.Commander, res imap.RespHandlerFrom) (status *imap.StatusResp, err error) {
	return c.execute(cmdr, res)
}

func (c *Client) handleContinuationReqs(continues chan bool) {
	hdlr := make(imap.RespHandler)
	c.addHandler(hdlr)
	defer c.removeHandler(hdlr)

	defer close(continues)

	for h := range hdlr {
		if _, ok := h.Resp.(*imap.ContinuationResp); ok {
			// Only accept if waiting for a continuation request
			select {
			case continues <- true:
				h.Accept()
			default:
				h.Reject()
			}
		} else {
			h.Reject()
		}
	}
}

func (c *Client) gotStatusCaps(args []interface{}) {
	c.Caps = map[string]bool{}
	for _, cap := range args {
		c.Caps[cap.(string)] = true
	}
}

// The server can send unilateral data. This function handles it.
func (c *Client) handleUnilateral() {
	hdlr := make(imap.RespHandler)
	c.addHandler(hdlr)
	defer c.removeHandler(hdlr)

	greeted := make(chan struct{})

	// Make sure to start reading after we have set up the base handlers,
	// otherwise some messages will be lost.
	go c.read(greeted)

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

			if greeted != nil {
				switch res.Type {
				case imap.StatusPreauth:
					c.State = imap.AuthenticatedState
				case imap.StatusBye:
					c.State = imap.LogoutState
				case imap.StatusOk:
					c.State = imap.NotAuthenticatedState
				default:
					c.ErrorLog.Println("invalid greeting: ", res.Type)
					c.State = imap.LogoutState
				}

				if res.Code == imap.CodeCapability {
					c.gotStatusCaps(res.Arguments)
				}

				close(greeted)
				greeted = nil
			}

			switch res.Type {
			case imap.StatusOk:
				select {
				case c.Infos <- res:
				default:
				}
			case imap.StatusNo:
				select {
				case c.Warnings <- res:
				default:
				}
			case imap.StatusBad:
				select {
				case c.Errors <- res:
				default:
				}
			case imap.StatusBye:
				c.State = imap.LogoutState
				c.Mailbox = nil
				c.conn.Close()

				select {
				case c.Byes <- res:
				default:
				}
			}
		case *imap.Resp:
			if len(res.Fields) < 2 {
				h.Reject()
				break
			}

			name, ok := res.Fields[1].(string)
			if !ok || (name != "EXISTS" && name != "RECENT" && name != "EXPUNGE") {
				h.Reject()
				break
			}
			h.Accept()

			switch name {
			case "EXISTS":
				if c.Mailbox == nil {
					break
				}
				c.Mailbox.Messages, _ = imap.ParseNumber(res.Fields[0])

				select {
				case c.MailboxUpdates <- c.Mailbox:
				default:
				}
			case "RECENT":
				if c.Mailbox == nil {
					break
				}
				c.Mailbox.Recent, _ = imap.ParseNumber(res.Fields[0])

				select {
				case c.MailboxUpdates <- c.Mailbox:
				default:
				}
			case "EXPUNGE":
				seqNum, _ := imap.ParseNumber(res.Fields[0])

				select {
				case c.Expunges <- seqNum:
				default:
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

// Get an imap.Writer for this client's connection.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (c *Client) Writer() *imap.Writer {
	return c.conn.Writer
}

// Check if this client's connection has TLS enabled.
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
		conn:           imap.NewConn(conn, r, w),
		handlersLocker: &sync.Mutex{},
		greeted:        make(chan struct{}),
		loggedOut:      make(chan struct{}),
		State:          imap.ConnectingState,
		ErrorLog:       log.New(os.Stderr, "imap/client: ", log.LstdFlags),
	}

	go c.handleContinuationReqs(continues)
	go c.handleUnilateral()

	// greeting := c.handleGreeting()
	// greeting.Err()
	<-c.greeted
	return
}

// Connect to an IMAP server using an unencrypted connection.
func Dial(addr string) (c *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}

	c, err = New(conn)
	return
}

// Connect to an IMAP server using an encrypted connection.
func DialTLS(addr string, tlsConfig *tls.Config) (c *Client, err error) {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return
	}

	c, err = New(conn)
	c.isTLS = true
	return
}
