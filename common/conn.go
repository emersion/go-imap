package common

import (
	"bufio"
	"net"
	"sync"

	"io"
	"os"
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

	waiter *sync.WaitGroup

	// Set to true to print all commands and responses to STDOUT.
	Debug bool
}

func (c *Conn) init() {
	r := io.Reader(c.Conn)
	w := io.Writer(c.Conn)

	if c.Debug {
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
