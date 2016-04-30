package client

import (
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
}

func (c *Client) read() (err error) {
	// TODO: optimize readers, do not create new ones for each response
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		line := scanner.Text() + "\n"
		r := strings.NewReader(line)

		var res interface{}
		res, err = imap.ReadResp(imap.NewReader(bufio.NewReader(r)))
		if err != nil {
			return
		}

		for _, hdlr := range c.handlers {
			if hdlr == nil {
				continue
			}

			h := &imap.RespHandling{
				Resp: res,
				Accepts: make(chan bool),
			}

			hdlr <- h

			if <-h.Accepts {
				break
			}
		}
	}

	return scanner.Err()
}

func (c *Client) execute(cmdr imap.Commander, res imap.RespHandlerFrom) (err error) {
	cmd := cmdr.Command()

	_, err = cmd.WriteTo(c.conn)
	if err != nil {
		return
	}

	statusHdlr := make(imap.RespHandler)
	c.handlers = append(c.handlers, statusHdlr)

	defer (func() {
		close(statusHdlr)

		// TODO: really remove handler from array? (needs locker)
		for i, h := range c.handlers {
			if h == statusHdlr {
				c.handlers[i] = nil
			}
		}
	})()

	var hdlr imap.RespHandler
	if res != nil {
		hdlr := make(imap.RespHandler)
		defer close(hdlr)

		go (func() {
			err = res.HandleFrom(hdlr)
		})()
	}

	for h := range statusHdlr {
		if status, ok := h.Resp.(*imap.StatusResp); ok && status.Tag == cmd.Tag {
			h.Accept()
			err = status
			return
		} else if hdlr != nil {
			hdlr <- h
		} else {
			h.Reject()
		}
	}

	return
}

func (c *Client) handleCaps() (err error) {
	res := &responses.Capability{}

	hdlr := make(imap.RespHandler)
	c.handlers = append(c.handlers, hdlr)
	defer close(hdlr)

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

	err = c.execute(cmd, nil)
	caps = c.Caps
	return
}

func (c *Client) StartTLS(tlsConfig *tls.Config) (err error) {
	if _, ok := c.conn.(*tls.Conn); ok {
		err = errors.New("TLS is already enabled")
		return
	}

	cmd := &commands.StartTLS{}

	err = c.execute(cmd, nil)
	if err != nil {
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

func NewClient(conn net.Conn) *Client {
	c := &Client{
		conn: conn,
	}

	go c.read()
	go c.handleCaps()

	return c
}

func Dial(addr string) (c *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}

	c = NewClient(conn)
	return
}

func DialTLS(addr string, tlsConfig *tls.Config) (c *Client, err error) {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return
	}

	c = NewClient(conn)
	return
}
