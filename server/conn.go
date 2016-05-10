package server

import (
	"bufio"
	"crypto/tls"
	"net"
	"os"
	"io"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/backend"
)

type Conn struct {
	*common.Reader
	*common.Writer

	conn net.Conn
	continues chan bool

	// This connection's server.
	Server *Server
	// This connection's current state.
	State common.ConnState
	// If the client is logged in, the user.
	User backend.User
	// If the client has selected a mailbox, the mailbox.
	Mailbox backend.Mailbox
	// True if the currently selected mailbox has been opened in read-only mode.
	MailboxReadOnly bool
}

// Close this connection.
func (c *Conn) Close() error {
	if err := c.conn.Close(); err != nil {
		return err
	}

	close(c.continues)

	c.State = common.LogoutState
	return nil
}

func (c *Conn) getCaps() (caps []string) {
	caps = []string{"IMAP4rev1"}

	if c.State == common.NotAuthenticatedState {
		if !c.IsTLS() && c.Server.TLSConfig != nil {
			caps = append(caps, "STARTTLS")
		}

		if !c.CanAuth() {
			caps = append(caps, "LOGINDISABLED")
		} else {
			caps = append(caps, "AUTH=PLAIN")
		}
	}

	caps = append(caps, c.Server.getCaps(c.State)...)
	return
}

func (c *Conn) sendContinuationReqs() {
	for range c.continues {
		cont := &common.ContinuationResp{Info: "send literal"}
		cont.WriteTo(c.Writer)
	}
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
	continues := make(chan bool)
	tee := io.TeeReader(c, os.Stdout)
	r := common.NewServerReader(bufio.NewReader(tee), continues)
	w := common.NewWriter(c)

	conn := &Conn{
		Reader: r,
		Writer: w,

		conn: c,
		continues: continues,

		Server: s,
		State: common.NotAuthenticatedState,
	}

	go conn.sendContinuationReqs()

	return conn
}
