package server

import (
	"bufio"
	"crypto/tls"
	"net"

	imap "github.com/emersion/imap/common"
)

type Conn struct {
	*imap.Reader
	*imap.Writer

	conn net.Conn

	Server *Server
	State imap.ConnState
}

func (c *Conn) Close() error {
	if err := c.conn.Close(); err != nil {
		return err
	}

	c.State = imap.LogoutState

	return nil
}

func (c *Conn) getCaps() (caps []string) {
	caps = []string{"IMAP4rev1"}

	switch c.State {
	case imap.NotAuthenticatedState:
		if !c.CanAuth() {
			caps = append(caps, imap.StartTLS, "LOGINDISABLED")
			return
		}

		caps = append(caps, "AUTH=PLAIN")
	case imap.AuthenticatedState, imap.SelectedState:
	}

	return
}

func (c *Conn) greet() error {
	caps := c.getCaps()
	args := make([]interface{}, len(caps))
	for i, cap := range caps {
		args[i] = cap
	}

	greeting := &imap.StatusResp{
		Tag: "*",
		Type: imap.OK,
		Code: imap.Capability,
		Arguments: args,
		Info: "IMAP4rev1 Service Ready",
	}

	return greeting.WriteTo(c.Writer)
}

func (c *Conn) IsTLS() bool {
	_, ok := c.conn.(*tls.Conn)
	return ok
}

func (c *Conn) CanAuth() bool {
	return c.IsTLS() || c.Server.AllowInsecureAuth
}

func newConn(s *Server, c net.Conn) *Conn {
	r := imap.NewReader(bufio.NewReader(c))
	w := imap.NewWriter(c)

	return &Conn{
		Reader: r,
		Writer: w,

		conn: c,

		Server: s,
		State: imap.NotAuthenticatedState,
	}
}
