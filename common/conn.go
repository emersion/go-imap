package common

import (
	"bufio"
	"net"
	"sync"
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

	waiter *sync.WaitGroup
}

func (c *Conn) init() {
	//tee := io.TeeReader(c.Conn, os.Stdout)
	c.Reader.reader = bufio.NewReader(c.Conn)
	c.Writer.writer = c.Conn
}

// Upgrade a connection, e.g. wrap an unencrypted connection with an encrypted
// tunnel.
func (c *Conn) Upgrade(upgrader ConnUpgrader) error {
	// Block reads and writes during the upgrading process
	c.waiter = &sync.WaitGroup{}
	c.waiter.Add(1)

	upgraded, err := upgrader(c.Conn)
	if err != nil {
		return err
	}

	c.Conn = upgraded
	c.init()

	c.waiter.Done()
	c.waiter = nil
	return nil
}

// Wait for the connection to be ready for reads and writes.
func (c *Conn) Wait() {
	if c.waiter != nil {
		c.waiter.Wait()
	}
}

// Create a new IMAP connection.
func NewConn(conn net.Conn, r *Reader, w *Writer) *Conn {
	c := &Conn{Conn: conn, Reader: r, Writer: w}

	c.init()
	return c
}
