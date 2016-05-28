package common

import (
	"bufio"
	"net"
)

// A function that upgrades a connection.
//
// This should only be used by libraries implementing an IMAP extension (e.g.
// COMPRESS).
type ConnUpgrader func(conn net.Conn) (net.Conn, error)

// An interface implemented by net.Conn that allows to flush buffered data to
// the remote.
type FlushableConn interface {
	net.Conn

	// Flush sends any buffered data to the client.
	Flush() error
}

// An IMAP connection.
type Conn struct {
	net.Conn
	*Reader
	*Writer
}

func (c *Conn) init() {
	c.Reader.reader = bufio.NewReader(c.Conn)
	c.Writer.writer = c.Conn
}

// Upgrade a connection, e.g. wrap an unencrypted connection with an encrypted
// tunnel.
func (c *Conn) Upgrade(upgrader ConnUpgrader) error {
	upgraded, err := upgrader(c.Conn)
	if err != nil {
		return err
	}

	c.Conn = upgraded
	c.init()
	return nil
}

// Create a new IMAP connection.
func NewConn(conn net.Conn, r *Reader, w *Writer) *Conn {
	c := &Conn{Conn: conn, Reader: r, Writer: w}

	c.init()
	return c
}
