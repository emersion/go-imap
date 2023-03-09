package imapwire

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// An Encoder writes IMAP data.
//
// Most methods don't return an error, instead they defer error handling until
// CRLF is called. These methods return the Encoder so that calls can be
// chained.
type Encoder struct {
	w       *bufio.Writer
	err     error
	literal bool
}

// NewEncoder creates a new encoder.
func NewEncoder(w *bufio.Writer) *Encoder {
	return &Encoder{w: w}
}

func (enc *Encoder) writeString(s string) *Encoder {
	if enc.err != nil {
		return enc
	}
	if enc.literal {
		enc.err = fmt.Errorf("imapwire: cannot encode while a literal is open")
		return enc
	}
	if _, err := enc.w.WriteString(s); err != nil {
		enc.err = err
	}
	return enc
}

// CRLF writes a "\r\n" sequence and flushes the buffered writer.
func (enc *Encoder) CRLF() error {
	enc.writeString("\r\n")
	if enc.err != nil {
		return enc.err
	}
	return enc.w.Flush()
}

func (enc *Encoder) Atom(s string) *Encoder {
	return enc.writeString(s)
}

func (enc *Encoder) SP() *Encoder {
	return enc.writeString(" ")
}

func (enc *Encoder) Special(ch byte) *Encoder {
	return enc.writeString(string(ch))
}

func (enc *Encoder) String(s string) *Encoder {
	// TODO: if the string contains CR/LF, use a literal
	var sb strings.Builder
	sb.Grow(2 + len(s))
	sb.WriteByte('"')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' || ch == '\\' {
			sb.WriteByte('\\')
		}
		sb.WriteByte(ch)
	}
	sb.WriteByte('"')
	return enc.writeString(sb.String())
}

func (enc *Encoder) Mailbox(name string) *Encoder {
	if strings.EqualFold(name, "INBOX") {
		return enc.Atom("INBOX")
	} else {
		return enc.String(name)
	}
}

func (enc *Encoder) Number(v uint32) *Encoder {
	return enc.writeString(strconv.FormatUint(uint64(v), 10))
}

func (enc *Encoder) Number64(v int64) *Encoder {
	// TODO: disallow negative values
	return enc.writeString(strconv.FormatInt(v, 10))
}

// List writes a parenthesized list.
func (enc *Encoder) List(n int, f func(i int)) {
	enc.Special('(')
	for i := 0; i < n; i++ {
		if i > 0 {
			enc.SP()
		}
		f(i)
	}
	enc.Special(')')
}

// Literal writes a literal.
//
// The caller must write exactly size bytes to the returned writer.
//
// If sync is non-nil, the literal is synchronizing: the encoder will wait for
// nil to be sent to the channel before writing the literal data. If an error
// is sent to the channel, the literal will be cancelled.
func (enc *Encoder) Literal(size int64, sync <-chan error) io.WriteCloser {
	// TODO: literal8
	enc.writeString("{")
	enc.Number64(size)
	if sync == nil {
		enc.writeString("+")
	}
	enc.writeString("}")

	if sync == nil {
		enc.writeString("\r\n")
	} else {
		if err := enc.CRLF(); err != nil {
			return errorWriter{err}
		}
		err, ok := <-sync
		if !ok {
			err = fmt.Errorf("imapwire: literal cancelled")
		}
		if err != nil {
			if enc.err == nil {
				enc.err = err
			}
			return errorWriter{err}
		}
	}

	enc.literal = true
	return &literalWriter{
		enc: enc,
		n:   size,
	}
}

type errorWriter struct {
	err error
}

func (ew errorWriter) Write(b []byte) (int, error) {
	return 0, ew.err
}

func (ew errorWriter) Close() error {
	return ew.err
}

type literalWriter struct {
	enc *Encoder
	n   int64
}

func (lw *literalWriter) Write(b []byte) (int, error) {
	if lw.n-int64(len(b)) < 0 {
		return 0, fmt.Errorf("wrote too many bytes in literal")
	}
	n, err := lw.enc.w.Write(b)
	lw.n -= int64(n)
	return n, err
}

func (lw *literalWriter) Close() error {
	lw.enc.literal = false
	if lw.n != 0 {
		return fmt.Errorf("wrote too few bytes in literal (%v remaining)", lw.n)
	}
	return nil
}
