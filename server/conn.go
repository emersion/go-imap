package server

import (
	"crypto/tls"
	"log"
	"net"
	"sync"

	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/backend"
)

type Conn interface {
	// Get this connection's server.
	Server() *Server
	// Get this connection's context.
	Context() *Context
	// Get a list of capabilities enabled for this connection.
	Capability() []string
	// Write a response to this connection.
	WriteResp(res common.WriterTo) error
	// Check if TLS is enabled on this connection.
	IsTLS() bool
	// Upgrade a connection, e.g. wrap an unencrypted connection with an encrypted
	// tunnel.
	Upgrade(upgrader common.ConnUpgrader) error
	// Close this connection.
	Close() error

	conn() *conn
}

type Context struct {
	// This connection's current state.
	State common.ConnState
	// If the client is logged in, the user.
	User backend.User
	// If the client has selected a mailbox, the mailbox.
	Mailbox backend.Mailbox
	// True if the currently selected mailbox has been opened in read-only mode.
	MailboxReadOnly bool
}

type conn struct {
	*common.Conn

	s *Server
	ctx *Context
	isTLS bool
	continues chan bool
	silent bool
	locker sync.Locker
}

func newConn(s *Server, c net.Conn) *conn {
	continues := make(chan bool)
	r := common.NewServerReader(nil, continues)
	w := common.NewWriter(nil)

	_, isTLS := c.(*tls.Conn)

	conn := &conn{
		Conn: common.NewConn(c, r, w),

		s: s,
		ctx: &Context{
			State: common.NotAuthenticatedState,
		},
		isTLS: isTLS,
		continues: continues,
		locker: &sync.Mutex{},
	}

	go conn.sendContinuationReqs()

	return conn
}

func (c *conn) conn() *conn {
	return c
}

func (c *conn) Server() *Server {
	return c.s
}

func (c *conn) Context() *Context {
	return c.ctx
}

// Write a response to this connection.
func (c *conn) WriteResp(res common.WriterTo) error {
	c.locker.Lock()
	defer c.locker.Unlock()

	if err := res.WriteTo(c.Writer); err != nil {
		return err
	}

	return c.Writer.Flush()
}

// Close this connection.
func (c *conn) Close() error {
	if c.ctx.User != nil {
		c.ctx.User.Logout()
	}

	if err := c.Conn.Close(); err != nil {
		return err
	}

	close(c.continues)

	c.ctx.State = common.LogoutState
	return nil
}

func (c *conn) Capability() (caps []string) {
	caps = []string{"IMAP4rev1"}

	if c.ctx.State == common.NotAuthenticatedState {
		if !c.IsTLS() && c.s.TLSConfig != nil {
			caps = append(caps, "STARTTLS")
		}

		if !c.canAuth() {
			caps = append(caps, "LOGINDISABLED")
		} else {
			caps = append(caps, "AUTH=PLAIN")
		}
	}

	caps = append(caps, c.s.Capability(c.ctx.State)...)
	return
}

func (c *conn) sendContinuationReqs() {
	for range c.continues {
		cont := &common.ContinuationResp{Info: "send literal"}
		if err := c.WriteResp(cont); err != nil {
			log.Println("WARN: cannot send continuation request:", err)
		}
	}
}

func (c *conn) greet() error {
	caps := c.Capability()
	args := make([]interface{}, len(caps))
	for i, cap := range caps {
		args[i] = cap
	}

	greeting := &common.StatusResp{
		Type: common.StatusOk,
		Code: common.CodeCapability,
		Arguments: args,
		Info: "IMAP4rev1 Service Ready",
	}

	return c.WriteResp(greeting)
}

// Check if this connection is encrypted.
func (c *conn) IsTLS() bool {
	return c.isTLS
}

// Check if the client can use plain text authentication.
func (c *conn) canAuth() bool {
	return c.IsTLS() || c.s.AllowInsecureAuth
}
