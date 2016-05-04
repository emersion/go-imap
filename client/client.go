// An IMAP client.
package client

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"sync"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/responses"
)

// An IMAP client.
type Client struct {
	conn net.Conn
	writer *imap.Writer

	handlers []imap.RespHandler
	handlersLocker sync.Locker

	// The server capabilities.
	Caps map[string]bool
	// The current connection state.
	State imap.ConnState
	// The selected mailbox, if there is one.
	Selected *imap.MailboxStatus
}

func (c *Client) read() error {
	r := imap.NewReader(bufio.NewReader(c.conn))

	defer (func () {
		c.handlersLocker.Lock()
		defer c.handlersLocker.Unlock()

		for _, hdlr := range c.handlers {
			close(hdlr)
		}
		c.handlers = nil
	})()

	for {
		if c.State == imap.LogoutState {
			return nil
		}

		res, err := imap.ReadResp(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Println("Error reading response:", err)
			continue
		}

		c.handlersLocker.Lock()

		var accepted bool
		for _, hdlr := range c.handlers {
			if hdlr == nil {
				continue
			}

			h := &imap.RespHandling{
				Resp: res,
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
			log.Println("Response has not been handled", res)
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

	_, err = cmd.WriteTo(c.writer)
	if err != nil {
		return
	}

	statusHdlr := make(imap.RespHandler)
	c.addHandler(statusHdlr)
	defer c.removeHandler(statusHdlr)

	var hdlr imap.RespHandler
	var done chan error
	defer (func () {
		if hdlr != nil { close(hdlr) }
		if done != nil { close(done) }
	})()

	if res != nil {
		hdlr = make(imap.RespHandler)
		done = make(chan error)

		go (func() {
			err := res.HandleFrom(hdlr)
			done <- err
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

func (c *Client) handleContinuationReqs(continues chan bool) {
	hdlr := make(imap.RespHandler)
	c.addHandler(hdlr)
	defer c.removeHandler(hdlr)

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
	c.Caps = map[string]bool{}
	for _, cap := range args {
		c.Caps[cap.(string)] = true
	}
}

func (c *Client) handleGreeting() *imap.StatusResp {
	hdlr := make(imap.RespHandler)
	c.addHandler(hdlr)
	defer c.removeHandler(hdlr)

	// Make sure to start reading after we have set up the base handlers,
	// otherwise some messages will be lost.
	go c.read()

	for h := range hdlr {
		status, ok := h.Resp.(*imap.StatusResp)
		if !ok || status.Tag != "*" || (status.Type != imap.OK && status.Type != imap.PREAUTH && status.Type != imap.BYE) {
			h.Reject()
			continue
		}

		h.Accept()

		if status.Code == "CAPABILITY" {
			c.gotStatusCaps(status.Arguments)
		}

		if status.Type == imap.PREAUTH {
			c.State = imap.AuthenticatedState
		}
		if status.Type == imap.BYE {
			c.State = imap.LogoutState
		}

		go c.handleBye()

		return status
	}

	return nil
}

func (c *Client) handleBye() *imap.StatusResp {
	hdlr := make(imap.RespHandler)
	c.addHandler(hdlr)
	defer c.removeHandler(hdlr)

	for h := range hdlr {
		status, ok := h.Resp.(*imap.StatusResp)
		if !ok || status.Tag != "*" || status.Type != imap.BYE {
			h.Reject()
			continue
		}

		h.Accept()

		c.State = imap.LogoutState
		c.Selected = nil
		c.conn.Close()

		return status
	}

	return nil
}

func (c *Client) handleCaps() (err error) {
	res := &responses.Capability{}

	hdlr := make(imap.RespHandler)
	c.addHandler(hdlr)
	defer c.removeHandler(hdlr)

	for {
		err = res.HandleFrom(hdlr)
		if err != nil {
			return
		}

		c.Caps = map[string]bool{}
		for _, name := range res.Caps {
			c.Caps[name] = true
		}
	}

	return nil
}

// Create a new client from an existing connection.
func NewClient(conn net.Conn) (c *Client, err error) {
	c = &Client{
		conn: conn,
		handlersLocker: &sync.Mutex{},
		State: imap.NotAuthenticatedState,
	}

	continues := make(chan bool)
	c.writer = imap.NewClientWriter(c.conn, continues)

	go c.handleContinuationReqs(continues)
	go c.handleCaps()

	greeting := c.handleGreeting()
	greeting.Err()
	return
}

// Connect to an IMAP server using an unencrypted connection.
func Dial(addr string) (c *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}

	c, err = NewClient(conn)
	return
}

// Connect to an IMAP server using an encrypted connection.
func DialTLS(addr string, tlsConfig *tls.Config) (c *Client, err error) {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return
	}

	c, err = NewClient(conn)
	return
}
