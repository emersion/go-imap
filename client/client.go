package client

import (
	"errors"
	"io"
	"net"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
)

type Client struct {
	conn net.Conn

	capabilities []string
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

func (c *Client) Capability() []string {
	cmd := &commands.Capability{}
	res := &responses.Capability{}
	c.execute(cmd, res)
	return res.Capabilities
}

func NewClient(conn net.Conn) *Client {
	c := &Client{
		conn: conn,
	}

	return c
}
