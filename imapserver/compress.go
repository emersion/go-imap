package imapserver

import (
	"bytes"
	"compress/flate"
	"io"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) canCompress() bool {
	switch c.state {
	case imap.ConnStateAuthenticated, imap.ConnStateSelected:
		return true // TODO
	default:
		return false
	}
}

func (c *Conn) handleCompress(tag string, dec *imapwire.Decoder) error {
	var algo string
	if !dec.ExpectSP() || !dec.ExpectAtom(&algo) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	if !c.canCompress() {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "COMPRESS not available",
		}
	}
	if algo != "DEFLATE" {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "Unsupported compression algorithm",
		}
	}

	// Do not allow to write uncompressed data past this point: keep c.encMutex
	// locked until the end
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

	c.mutex.Lock()
	// TODO
	c.mutex.Unlock()

	w, err := flate.NewWriter(c.conn, flate.DefaultCompression)
	if err != nil {
		panic(err) // can only happen due to bad arguments
	}

	rw := c.server.options.wrapReadWriter(struct {
		io.Reader
		io.Writer
	}{
		Reader: flate.NewReader(r),
		Writer: w,
	})
	c.br.Reset(rw)
	c.bw.Reset(rw)

	return nil
}
