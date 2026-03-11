package imapclient

import (
	"bufio"
	"bytes"
	"compress/flate"
	"io"
)

// CompressOptions contains options for Client.Compress.
type CompressOptions struct{}

// Compress enables connection-level compression.
//
// Unlike other commands, this method blocks until the command completes.
//
// A nil options pointer is equivalent to a zero options value.
func (c *Client) Compress(options *CompressOptions) error {
	upgradeDone := make(chan struct{})
	cmd := &compressCommand{
		upgradeDone: upgradeDone,
	}
	enc := c.beginCommand("COMPRESS", cmd)
	enc.SP().Atom("DEFLATE")
	enc.flush()
	defer enc.end()

	// The client MUST NOT send any further commands until it has seen the
	// result of COMPRESS.

	if err := cmd.Wait(); err != nil {
		return err
	}

	// The decoder goroutine will invoke Client.upgradeCompress
	<-upgradeDone
	return nil
}

func (c *Client) upgradeCompress(compress *compressCommand) {
	defer close(compress.upgradeDone)

	// Drain buffered data from our bufio.Reader
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, c.br, int64(c.br.Buffered())); err != nil {
		panic(err) // unreachable
	}

	conn := c.conn
	if c.tlsConn != nil {
		conn = c.tlsConn
	}

	var r io.Reader
	if buf.Len() > 0 {
		r = io.MultiReader(&buf, conn)
	} else {
		r = c.conn
	}

	w, err := flate.NewWriter(conn, flate.DefaultCompression)
	if err != nil {
		panic(err) // can only happen due to bad arguments
	}

	rw := c.options.wrapReadWriter(struct {
		io.Reader
		io.Writer
	}{
		Reader: flate.NewReader(r),
		Writer: w,
	})

	c.br.Reset(rw)
	// Unfortunately we can't re-use the bufio.Writer here, it races with
	// Client.Compress
	c.bw = bufio.NewWriter(rw)
}

type compressCommand struct {
	cmd
	upgradeDone chan<- struct{}
}
