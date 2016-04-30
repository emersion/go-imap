package client

import (
	"log"
	"bufio"
	"errors"
	"net"
	"crypto/tls"
	"strings"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
)

type Client struct {
	conn net.Conn
	handlers []imap.RespHandler

	Caps map[string]bool
	State imap.ConnState
}

func (c *Client) read() (err error) {
	// TODO: optimize readers, do not create new ones for each response
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		line := scanner.Text()
		r := strings.NewReader(line + "\n")

		log.Println("S:", line)

		var res interface{}
		res, err = imap.ReadResp(imap.NewReader(bufio.NewReader(r)))
		if err != nil {
			log.Println("Error reading response:", err)
			continue
		}

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

		if !accepted {
			log.Println("Response has not been handled", res)
		}
	}

	return scanner.Err()
}

func (c *Client) addHandler(hdlr imap.RespHandler) {
	// TODO: needs locker
	c.handlers = append(c.handlers, hdlr)
}

func (c *Client) removeHandler(hdlr imap.RespHandler) {
	close(hdlr)

	// TODO: really remove handler from array? (needs locker)
	for i, h := range c.handlers {
		if h == hdlr {
			c.handlers[i] = nil
		}
	}
}

func (c *Client) execute(cmdr imap.Commander, res imap.RespHandlerFrom) (status *imap.StatusResp, err error) {
	cmd := cmdr.Command()
	cmd.Tag = generateTag()

	log.Println(cmd)

	_, err = cmd.WriteTo(c.conn)
	if err != nil {
		return
	}

	statusHdlr := make(imap.RespHandler)
	c.addHandler(statusHdlr)
	defer c.removeHandler(statusHdlr)

	var hdlr imap.RespHandler
	defer (func () {
		if hdlr != nil {
			close(hdlr)
		}
	})()

	if res != nil {
		hdlr = make(imap.RespHandler)

		go (func() {
			err = res.HandleFrom(hdlr)
		})()
	}

	for h := range statusHdlr {
		if status, ok := h.Resp.(*imap.StatusResp); ok && status.Tag == cmd.Tag {
			h.Accept()
			return status, nil
		} else if hdlr != nil {
			hdlr <- h
		} else {
			h.Reject()
		}
	}

	return
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

	for h := range hdlr {
		status, ok := h.Resp.(*imap.StatusResp)
		if !ok || status.Tag != "*" || (status.Type != imap.OK && status.Type != imap.PREAUTH && status.Type != imap.BYE) {
			h.Reject()
			continue
		}

		h.Accept()

		if status.Code == imap.CAPABILITY {
			c.gotStatusCaps(status.Arguments)
		}

		if status.Type == imap.PREAUTH {
			c.State = imap.AuthenticatedState
		}

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

func (c *Client) Capability() (caps map[string]bool, err error) {
	cmd := &commands.Capability{}

	_, err = c.execute(cmd, nil)
	caps = c.Caps
	return
}

func (c *Client) StartTLS(tlsConfig *tls.Config) (err error) {
	if _, ok := c.conn.(*tls.Conn); ok {
		err = errors.New("TLS is already enabled")
		return
	}

	cmd := &commands.StartTLS{}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}
	if err = status.Err(); err != nil {
		return
	}

	tlsConn := tls.Client(c.conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		return
	}

	c.conn = tlsConn
	return
}

func (c *Client) Login(username, password string) (err error) {
	if c.State != imap.NotAuthenticatedState {
		err = errors.New("Already logged in")
		return
	}
	if c.Caps["LOGINDISABLED"] {
		err = errors.New("Login is disabled in current state")
		return
	}

	cmd := &commands.Login{
		Username: username,
		Password: password,
	}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}
	if err = status.Err(); err != nil {
		return
	}

	if status.Code == imap.CAPABILITY {
		c.gotStatusCaps(status.Arguments)
	}

	c.State = imap.AuthenticatedState

	return
}

func (c *Client) List(ref, mbox string, ch chan<- *imap.MailboxInfo) (err error) {
	defer close(ch)

	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = errors.New("Not logged in")
		return
	}

	cmd := &commands.List{
		Reference: ref,
		Mailbox: mbox,
	}
	res := &responses.List{Mailboxes: ch}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}
	if err = status.Err(); err != nil {
		return
	}
	return
}

func NewClient(conn net.Conn) (c *Client, err error) {
	c = &Client{
		conn: conn,
		State: imap.NotAuthenticatedState,
	}

	go c.read()
	go c.handleCaps()

	greeting := c.handleGreeting()
	greeting.Err()
	return
}

func Dial(addr string) (c *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}

	c, err = NewClient(conn)
	return
}

func DialTLS(addr string, tlsConfig *tls.Config) (c *Client, err error) {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return
	}

	c, err = NewClient(conn)
	return
}
