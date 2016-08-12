package server

import (
	"crypto/tls"
	"io"
	"net"
	"sync"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
)

// A connection.
type Conn interface {
	io.Reader

	// Get this connection's server.
	Server() *Server
	// Get this connection's context.
	Context() *Context
	// Get a list of capabilities enabled for this connection.
	Capabilities() []string
	// Write a response to this connection.
	WriteResp(res imap.WriterTo) error
	// IsTLS returns true if TLS is enabled.
	IsTLS() bool
	// TLSState returns the TLS connection state if TLS is enabled, nil otherwise.
	TLSState() *tls.ConnectionState
	// Upgrade a connection, e.g. wrap an unencrypted connection with an encrypted
	// tunnel.
	Upgrade(upgrader imap.ConnUpgrader) error
	// Close this connection.
	Close() error

	conn() *imap.Conn
	reader() *imap.Reader
	writer() imap.Writer
	locker() sync.Locker
	greet() error
	setTLSConn(*tls.Conn)
	silent() *bool // TODO: remove this
}

// A connection's context.
type Context struct {
	// This connection's current state.
	State imap.ConnState
	// If the client is logged in, the user.
	User backend.User
	// If the client has selected a mailbox, the mailbox.
	Mailbox backend.Mailbox
	// True if the currently selected mailbox has been opened in read-only mode.
	MailboxReadOnly bool
}

type conn struct {
	*imap.Conn

	s         *Server
	ctx       *Context
	l         sync.Locker
	tlsConn   *tls.Conn
	continues chan bool
	silentVal bool
}

func newConn(s *Server, c net.Conn) *conn {
	continues := make(chan bool)
	r := imap.NewServerReader(nil, continues)
	w := imap.NewWriter(nil)

	tlsConn, _ := c.(*tls.Conn)

	conn := &conn{
		Conn: imap.NewConn(c, r, w),

		s: s,
		l: &sync.Mutex{},
		ctx: &Context{
			State: imap.NotAuthenticatedState,
		},
		tlsConn:   tlsConn,
		continues: continues,
	}

	go conn.sendContinuationReqs()

	return conn
}

func (c *conn) conn() *imap.Conn {
	return c.Conn
}

func (c *conn) reader() *imap.Reader {
	return c.Reader
}

func (c *conn) writer() imap.Writer {
	return c.Writer
}

func (c *conn) locker() sync.Locker {
	return c.l
}

func (c *conn) Server() *Server {
	return c.s
}

func (c *conn) Context() *Context {
	return c.ctx
}

// Write a response to this connection.
func (c *conn) WriteResp(res imap.WriterTo) error {
	c.l.Lock()
	defer c.l.Unlock()

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

	c.ctx.State = imap.LogoutState
	return nil
}

func (c *conn) Capabilities() (caps []string) {
	caps = c.s.Capabilities(c.ctx.State)

	if c.ctx.State == imap.NotAuthenticatedState {
		if !c.IsTLS() && c.s.TLSConfig != nil {
			caps = append(caps, "STARTTLS")
		}

		if !c.canAuth() {
			caps = append(caps, "LOGINDISABLED")
		} else {
			for name, _ := range c.s.auths {
				caps = append(caps, "AUTH="+name)
			}
		}
	}

	return
}

func (c *conn) sendContinuationReqs() {
	for range c.continues {
		cont := &imap.ContinuationResp{Info: "send literal"}
		if err := c.WriteResp(cont); err != nil {
			c.Server().ErrorLog.Println("cannot send continuation request: ", err)
		}
	}
}

func (c *conn) greet() error {
	caps := c.Capabilities()
	args := make([]interface{}, len(caps))
	for i, cap := range caps {
		args[i] = cap
	}

	greeting := &imap.StatusResp{
		Type:      imap.StatusOk,
		Code:      imap.CodeCapability,
		Arguments: args,
		Info:      "IMAP4rev1 Service Ready",
	}

	return c.WriteResp(greeting)
}

func (c *conn) setTLSConn(tlsConn *tls.Conn) {
	c.tlsConn = tlsConn
}

func (c *conn) IsTLS() bool {
	return c.tlsConn != nil
}

func (c *conn) TLSState() *tls.ConnectionState {
	if c.tlsConn != nil {
		state := c.tlsConn.ConnectionState()
		return &state
	}
	return nil
}

// Check if the client can use plain text authentication.
func (c *conn) canAuth() bool {
	return c.IsTLS() || c.s.AllowInsecureAuth
}

func (c *conn) silent() *bool {
	return &c.silentVal
}
