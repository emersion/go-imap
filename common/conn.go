package common

import (
	"bufio"
	"net"
	"io"
	"os"
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
	SelectedState = AuthenticatedState + 1 << 2

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

// An IMAP connection.
type Conn struct {
	net.Conn
	*Reader
	*Writer

	waits chan struct{}

	// Set to true to print all commands and responses to STDOUT.
	debug bool
}

func (c *Conn) init() {
	r := io.Reader(c.Conn)
	w := io.Writer(c.Conn)

	if c.debug {
		r = io.TeeReader(c.Conn, os.Stdout)
		w = io.MultiWriter(c.Conn, os.Stdout)
	}

	c.Reader.reader = bufio.NewReader(r)
	c.Writer.writer = bufio.NewWriter(w)
}

// Write any buffered data to the underlying connection.
func (c *Conn) Flush() (err error) {
	if err = c.Writer.Flush(); err != nil {
		return
	}

	if f, ok := c.Conn.(Flusher); ok {
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

// Wait for the connection to be ready for reads and writes.
func (c *Conn) Wait() {
	if c.waits != nil {
		<-c.waits
	}
}

// Enable or disable debugging.
func (c *Conn) SetDebug(debug bool) {
	c.debug = debug
	c.init()
}

// Create a new IMAP connection.
func NewConn(conn net.Conn, r *Reader, w *Writer) *Conn {
	c := &Conn{Conn: conn, Reader: r, Writer: w}

	c.init()
	return c
}
