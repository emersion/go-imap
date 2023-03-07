package imapwire

import (
	"bufio"
	"strings"
)

type Encoder struct {
	w   *bufio.Writer
	err error
}

func NewEncoder(w *bufio.Writer) *Encoder {
	return &Encoder{w: w}
}

func (enc *Encoder) writeString(s string) *Encoder {
	if enc.err != nil {
		return enc
	}
	if _, err := enc.w.WriteString(s); err != nil {
		enc.err = err
	}
	return enc
}

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
