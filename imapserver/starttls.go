package imapserver

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) canStartTLS() bool {
	_, isTLS := c.conn.(*tls.Conn)
	return c.server.TLSConfig != nil && c.state == imap.ConnStateNotAuthenticated && !isTLS
}

func (c *conn) handleStartTLS(tag string, dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if c.server.TLSConfig == nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "STARTTLS not supported",
		}
	}
	if !c.canStartTLS() {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "STARTTLS not available",
		}
	}

	err := c.writeStatusResp(tag, &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Text: "Begin TLS negotiation now",
	})
	if err != nil {
		return err
	}

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

	c.conn = tls.Server(cleartextConn, c.server.TLSConfig)
	c.br.Reset(c.conn)
	c.bw.Reset(c.conn)
	return nil
}

type startTLSConn struct {
	net.Conn
	r io.Reader
}

func (conn startTLSConn) Read(b []byte) (int, error) {
	return conn.r.Read(b)
}
