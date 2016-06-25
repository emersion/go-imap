package server

import (
	"crypto/tls"
	"net"

	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/backend"
)

type Conn struct {
	*common.Conn

	isTLS bool
	continues chan bool
	silent bool

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

// Write a response to this connection.
func (c *Conn) WriteRes(res common.WriterTo) error {
	if err := res.WriteTo(c.Writer); err != nil {
		return err
	}

	return c.Writer.Flush()
}

// Close this connection.
func (c *Conn) Close() error {
	if err := c.Conn.Close(); err != nil {
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

	return c.WriteRes(greeting)
}

// Check if this connection is encrypted.
func (c *Conn) IsTLS() bool {
	return c.isTLS
}

// Check if the client can use plain text authentication.
func (c *Conn) CanAuth() bool {
	return c.IsTLS() || c.Server.AllowInsecureAuth
}

func newConn(s *Server, c net.Conn) *Conn {
	continues := make(chan bool)
	r := common.NewServerReader(nil, continues)
	w := common.NewWriter(nil)

	_, isTLS := c.(*tls.Conn)

	conn := &Conn{
		Conn: common.NewConn(c, r, w),

		isTLS: isTLS,
		continues: continues,

		Server: s,
		State: common.NotAuthenticatedState,
	}

	go conn.sendContinuationReqs()

	return conn
}
