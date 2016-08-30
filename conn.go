package imap

import (
	"bufio"
	"io"
	"net"
)

// A connection state.
// See RFC 3501 section 3.
type ConnState int

const (
	// In the not authenticated state, the client MUST supply
	// authentication credentials before most commands will be
	// permitted.  This state is entered when a connection starts
	// unless the connection has been pre-authenticated.
	NotAuthenticatedState ConnState = 1

	// In the authenticated state, the client is authenticated and MUST
	// select a mailbox to access before commands that affect messages
	// will be permitted.  This state is entered when a
	// pre-authenticated connection starts, when acceptable
	// authentication credentials have been provided, after an error in
	// selecting a mailbox, or after a successful CLOSE command.
	AuthenticatedState = 1 << 1

	// In a selected state, a mailbox has been selected to access.
	// This state is entered when a mailbox has been successfully
	// selected.
	SelectedState = AuthenticatedState + 1<<2

	// In the logout state, the connection is being terminated. This
	// state can be entered as a result of a client request (via the
	// LOGOUT command) or by unilateral action on the part of either
	// the client or server.
	LogoutState = 0

	// ConnectedState is any state except LogoutState.
	ConnectedState = NotAuthenticatedState | AuthenticatedState | SelectedState
)

// A function that upgrades a connection.
//
// This should only be used by libraries implementing an IMAP extension (e.g.
// COMPRESS).
type ConnUpgrader func(conn net.Conn) (net.Conn, error)

type debugWriter struct {
	io.Writer

	local io.Writer
	remote io.Writer
}

// NewDebugWriter creates a new io.Writer that will write local network activity
// to local and remote network activity to remote.
func NewDebugWriter(local, remote io.Writer) io.Writer {
	return &debugWriter{Writer: local, local: local, remote: remote}
}

// An IMAP connection.
type Conn struct {
	net.Conn
	*Reader
	*Writer

	waits chan struct{}

	// Print all commands and responses to this io.Writer.
	debug io.Writer
}

// NewConn creates a new IMAP connection.
func NewConn(conn net.Conn, r *Reader, w *Writer) *Conn {
	c := &Conn{Conn: conn, Reader: r, Writer: w}

	c.init()
	return c
}

func (c *Conn) init() {
	r := io.Reader(c.Conn)
	w := io.Writer(c.Conn)

	if c.debug != nil {
		localDebug, remoteDebug := c.debug, c.debug
		if debug, ok := c.debug.(*debugWriter); ok {
			localDebug, remoteDebug = debug.local, debug.remote
		}

		r = io.TeeReader(c.Conn, remoteDebug)
		w = io.MultiWriter(c.Conn, localDebug)
	}

	c.Reader.reader = bufio.NewReader(r)
	c.Writer.Writer = bufio.NewWriter(w)
}

// Write implements io.Writer.
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.Writer.Write(b)
}

// Flush writes any buffered data to the underlying connection.
func (c *Conn) Flush() (err error) {
	if err = c.Writer.Flush(); err != nil {
		return
	}

	f, ok := c.Conn.(interface {
		Flush() error
	})
	if ok {
		if err = f.Flush(); err != nil {
			return
		}
	}

	return
}

// Upgrade a connection, e.g. wrap an unencrypted connection with an encrypted
// tunnel.
func (c *Conn) Upgrade(upgrader ConnUpgrader) error {
	// Block reads and writes during the upgrading process
	c.waits = make(chan struct{})
	defer close(c.waits)

	upgraded, err := upgrader(c.Conn)
	if err != nil {
		return err
	}

	c.Conn = upgraded
	c.init()
	return nil
}

// Wait waits for the connection to be ready for reads and writes.
func (c *Conn) Wait() {
	if c.waits != nil {
		<-c.waits
	}
}

// SetDebug defines an io.Writer to which all network activity will be logged.
// If nil is provided, network activity will not be logged.
func (c *Conn) SetDebug(w io.Writer) {
	c.debug = w
	c.init()
}
