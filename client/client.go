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

func (c *Client) execute(cmdr imap.Commander, r imap.RespHandlerFrom) (err error) {
	cmd := cmdr.Command()

	_, err = cmd.WriteTo(c.conn)
	if err != nil {
		return
	}

	statusHdlr := make(imap.RespHandler)
	defer close(statusHdlr)
	c.handlers = append(c.handlers, statusHdlr)

	var hdlr imap.RespHandler
	done := make(chan bool)
	if r != nil {
		hdlr := make(imap.RespHandler)
		defer close(hdlr)

		go (func() {
			err = r.HandleFrom(hdlr)
			done <- true
		})()
	}

	for {
		select {
		case h := <-statusHdlr:
			if status, ok := h.Resp.(*imap.StatusResp); ok && status.Tag == cmd.Tag {
				h.Accept()
				err = status
				return
			} else if hdlr != nil {
				hdlr <- h
			} else {
				h.Reject()
			}
		case <-done:
			return
		}
	}
	return
}

func (c *Client) Capability() (res *responses.Capability, err error) {
	cmd := &commands.Capability{}
	res = &responses.Capability{}

	err = c.execute(cmd, res)
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
