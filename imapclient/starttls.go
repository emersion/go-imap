package imapclient

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
)

// startTLS sends a STARTTLS command.
//
// Unlike other commands, this method blocks until the command completes.
func (c *Client) startTLS(config *tls.Config) error {
	upgradeDone := make(chan struct{})
	cmd := &startTLSCommand{
		tlsConfig:   config,
		upgradeDone: upgradeDone,
	}
	enc := c.beginCommand("STARTTLS", cmd)
	enc.flush()
	defer enc.end()

	// Once a client issues a STARTTLS command, it MUST NOT issue further
	// commands until a server response is seen and the TLS negotiation is
	// complete

	if err := cmd.Wait(); err != nil {
		return err
	}

	// The decoder goroutine will invoke Client.upgradeStartTLS
	<-upgradeDone
	return nil
}

func (c *Client) upgradeStartTLS(tlsConfig *tls.Config) {
	// Drain buffered data from our bufio.Reader
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, c.br, int64(c.br.Buffered())); err != nil {
		panic(err) // unreachable
	}

	var cleartextConn net.Conn
	if buf.Len() > 0 {
		r := io.MultiReader(&buf, c.conn)
		cleartextConn = startTLSConn{c.conn, r}
	} else {
		cleartextConn = c.conn
	}

	tlsConn := tls.Client(cleartextConn, tlsConfig)
	rw := c.options.wrapReadWriter(tlsConn)

	c.br.Reset(rw)
	// Unfortunately we can't re-use the bufio.Writer here, it races with
	// Client.StartTLS
	c.bw = bufio.NewWriter(rw)
}

type startTLSCommand struct {
	cmd
	tlsConfig   *tls.Config
	upgradeDone chan<- struct{}
}

type startTLSConn struct {
	net.Conn
	r io.Reader
}

func (conn startTLSConn) Read(b []byte) (int, error) {
	return conn.r.Read(b)
}
