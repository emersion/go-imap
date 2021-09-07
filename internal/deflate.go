package internal

import (
	"compress/flate"
	"io"
	"net"
)

type deflateConn struct {
	net.Conn

	r io.ReadCloser
	w *flate.Writer
}

func (c *deflateConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func (c *deflateConn) Write(b []byte) (int, error) {
	return c.w.Write(b)
}

type flusher interface {
	Flush() error
}

func (c *deflateConn) Flush() error {
	if f, ok := c.Conn.(flusher); ok {
		if err := f.Flush(); err != nil {
			return err
		}
	}

	return c.w.Flush()
}

func (c *deflateConn) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}

	if err := c.w.Close(); err != nil {
		return err
	}

	return c.Conn.Close()
}

func CreateDeflateConn(c net.Conn, level int) (net.Conn, error) {
	r := flate.NewReader(c)
	w, err := flate.NewWriter(c, level)
	if err != nil {
		return nil, err
	}

	return &deflateConn{
		Conn: c,
		r:    r,
		w:    w,
	}, nil
}
