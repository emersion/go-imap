package client

import (
	"errors"
	"io"
	"net"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
)

type Client struct {
	conn net.Conn

	capabilities []string
	readers map[string]io.ReaderFrom
}

// TODO: support async reads
func (c *Client) execute(cmdr imap.Commander) (r io.ReaderFrom, err error) {
	cmd := cmdr.Command()

	r, ok := c.readers[cmd.Name]
	if !ok {
		// TODO: use generic reader?
		err = errors.New("No reader found for command: " + cmd.Name)
		return
	}

	_, err := cmd.WriteTo(c.conn)
	if err != nil {
		return err
	}

	_, err = r.ReadFrom(c.conn)
	return
}

func (c *Client) Capability() {
	cmd := &commands.Capability{}
	c.execute(cmd)
}

func NewClient(conn net.Conn) *Client {
	c := &Client{
		conn: conn,
	}

	return c
}
