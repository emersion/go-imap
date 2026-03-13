package imapserver

import (
	"bytes"
	"compress/flate"
	"crypto/tls"
	"io"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) canCompress() bool {
	_, isTLS := c.conn.(*tls.Conn)
	return !c.compressed && (c.server.options.TLSConfig == nil || isTLS)
}

func (c *Conn) handleCompress(tag string, dec *imapwire.Decoder) error {
	var algorithm string
	if !dec.ExpectSP() || !dec.ExpectAtom(&algorithm) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	if algorithm != "DEFLATE" {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unsupported compression algorithm",
		}
	}

	if !c.canCompress() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "COMPRESS not available",
		}
	}

	// Keep c.encMutex locked until the upgrade is complete
	enc := newResponseEncoder(c)
	defer enc.end()

	err := writeStatusResp(enc.Encoder, tag, &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Text: "Begin compression now",
	})
	if err != nil {
		return err
	}

	// Drain buffered data from our bufio.Reader
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, c.br, int64(c.br.Buffered())); err != nil {
		panic(err) // unreachable
	}

	var r io.Reader
	if buf.Len() > 0 {
		r = io.MultiReader(&buf, c.conn)
	} else {
		r = c.conn
	}

	w, err := flate.NewWriter(c.conn, flate.DefaultCompression)
	if err != nil {
		panic(err) // can only happen due to bad arguments
	}

	rw := c.server.options.wrapReadWriter(struct {
		io.Reader
		io.Writer
	}{
		Reader: flate.NewReader(r),
		Writer: &compressSyncFlushWriter{w: w},
	})

	c.br.Reset(rw)
	c.bw.Reset(rw)
	c.compressed = true

	return nil
}

// compressSyncFlushWriter wraps a flate.Writer and calls Flush() after each
// Write(). This ensures that compressed data is sent to the underlying
// connection after bufio.Writer.Flush() pushes its buffer.
type compressSyncFlushWriter struct {
	w *flate.Writer
}

func (sw *compressSyncFlushWriter) Write(p []byte) (int, error) {
	n, err := sw.w.Write(p)
	if err != nil {
		return n, err
	}
	if err := sw.w.Flush(); err != nil {
		return n, err
	}
	return n, nil
}
