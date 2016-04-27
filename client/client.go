package client

import (
	"errors"
	"io"
	"net"
	"crypto/tls"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
)

type Client struct {
	conn net.Conn
}

func (c *Client) execute(cmdr imap.Commander, r io.ReaderFrom) (err error) {
	cmd := cmdr.Command()

	_, err := cmd.WriteTo(c.conn)
	if err != nil {
		return err
	}

	_, err = r.ReadFrom(c.conn)
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
	res := &responses.StartTLS{}

	err = c.execute(cmd, res)
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
