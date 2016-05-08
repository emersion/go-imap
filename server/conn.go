package server

import (
	"bufio"
	"crypto/tls"
	"net"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/backend"
)

type Conn struct {
	*common.Reader
	*common.Writer

	conn net.Conn

	// This connection's server.
	Server *Server
	// This connection's current state.
	State common.ConnState
	// If the client is logged in, the user.
	User backend.User
	// If the client has selected a mailbox, the mailbox.
	Mailbox backend.Mailbox
}

// Close this connection.
func (c *Conn) Close() error {
	if err := c.conn.Close(); err != nil {
		return err
	}

	c.State = common.LogoutState
	return nil
}

func (c *Conn) getCaps() (caps []string) {
	caps = []string{"IMAP4rev1"}

	switch c.State {
	case common.NotAuthenticatedState:
		if !c.CanAuth() {
			caps = append(caps, common.StartTLS, "LOGINDISABLED")
			return
		}

		caps = append(caps, "AUTH=PLAIN")
	case common.AuthenticatedState, common.SelectedState:
	}

	return
}

func (c *Conn) greet() error {
	caps := c.getCaps()
	args := make([]interface{}, len(caps))
	for i, cap := range caps {
		args[i] = cap
	}

	greeting := &common.StatusResp{
		Tag: "*",
		Type: common.OK,
		Code: common.Capability,
		Arguments: args,
		Info: "IMAP4rev1 Service Ready",
	}

	return greeting.WriteTo(c.Writer)
}

// Check if this connection is encrypted.
func (c *Conn) IsTLS() bool {
	_, ok := c.conn.(*tls.Conn)
	return ok
}

// Check if the client can use plain text authentication.
func (c *Conn) CanAuth() bool {
	return c.IsTLS() || c.Server.AllowInsecureAuth
}

func newConn(s *Server, c net.Conn) *Conn {
	r := common.NewReader(bufio.NewReader(c))
	w := common.NewWriter(c)

	return &Conn{
		Reader: r,
		Writer: w,

		conn: c,

		Server: s,
		State: common.NotAuthenticatedState,
	}
}
