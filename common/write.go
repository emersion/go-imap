package common

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

// A string writer.
type StringWriter interface {
	// WriteString writes a string. It returns the number of bytes written. If the
	// count is less than len(s), it also returns an error explaining why the
	// write is short.
	// See https://golang.org/pkg/bufio/#writeString
	WriteString(s string) (int, error)
}

type writer interface {
	io.Writer
}

// An IMAP writer.
type Writer struct {
	writer

	continues <-chan bool
}

func (w *Writer) writeString(s string) (int, error) {
	return io.WriteString(w.writer, s)
}

// Write a separator.
func (w *Writer) WriteSp() (int, error) {
	return w.writeString(string(sp))
}

// Write a CRLF.
func (w *Writer) WriteCrlf() (int, error) {
	return w.writeString(string(cr) + string(lf))
}

// Write NIL.
func (w *Writer) WriteNil() (int, error) {
	return w.writeString("NIL")
}

func (w *Writer) WriteNumber(num uint32) (int, error) {
	return w.writeString(strconv.Itoa(int(num)))
}

func (w *Writer) writeAtomString(s string) (int, error) {
	return w.writeString(s)
}

func (w *Writer) writeQuotedString(s string) (int, error) {
	return w.writeString(string(dquote) + s + string(dquote))
}

func (w *Writer) WriteString(s string) (int, error) {
	if s == "NIL" || s == "" || strings.ContainsAny(s, " \"(") {
		return w.writeQuotedString(s)
	}
	return w.writeAtomString(s)
}

func (w *Writer) WriteDate(date *time.Time) (int, error) {
	if date == nil {
		return w.WriteNil()
	}
	return w.writeQuotedString(date.Format("2-Jan-2006 15:04:05 -0700"))
}

func (w *Writer) WriteFields(fields []interface{}) (N int, err error) {
	var n int

	for i, field := range fields {
		// Write separator
		if i > 0 {
			if n, err = w.WriteSp(); err != nil {
				return
			}
			N += n
		}

		if field == nil {
			n, err = w.WriteNil()
		} else {
			switch f := field.(type) {
			case string:
				n, err = w.WriteString(f)
			case int:
				n, err = w.WriteNumber(uint32(f))
			case uint32:
				n, err = w.WriteNumber(f)
			case *Literal:
				n, err = w.WriteLiteral(f)
			case []interface{}:
				n, err = w.WriteList(f)
			case *time.Time:
				n, err = w.WriteDate(f)
			case *SeqSet:
				n, err = w.writeString(f.String())
			case *BodySectionName:
				n, err = w.writeString(f.String())
			default:
				err = errors.New("Cannot format argument #" + strconv.Itoa(i))
			}
		}

		N += n
		if err != nil {
			return
		}
	}

	return
}

func (w *Writer) WriteList(fields []interface{}) (N int, err error) {
	n, err := w.writeString(string(listStart))
	N += n
	if err != nil {
		return
	}

	n, err = w.WriteFields(fields)
	N += n
	if err != nil {
		return
	}

	n, err = w.writeString(string(listEnd))
	N += n
	return
}

func (w *Writer) writeLiteralField(literal *Literal) (N int, err error) {
	field := string(literalStart) + strconv.Itoa(literal.Len()) + string(literalEnd)
	n, err := w.writeString(field)
	N += n
	if err != nil {
		return
	}

	n, err = w.WriteCrlf()
	N += n
	return
}

func (w *Writer) WriteLiteral(literal *Literal) (N int, err error) {
	if literal == nil {
		return w.WriteNil()
	}

	n, err := w.writeLiteralField(literal)
	N += n
	if err != nil {
		return
	}

	// If a channel is available, wait for a continuation request before sending data
	if w.continues != nil {
		if !<-w.continues {
			err = errors.New("Cannot send literal: no continuation request received")
			return
		}
	}

	n, err = w.Write(literal.Bytes())
	N += n
	return
}

func (w *Writer) WriteRespCode(code string, args []interface{}) (N int, err error) {
	n, err := w.writeString(string(respCodeStart))
	if err != nil {
		return
	}
	N += n

	fields := []interface{}{code}
	fields = append(fields, args...)

	if n, err = w.WriteFields(fields); err != nil {
		return
	}
	N += n

	n, err = w.writeString(string(respCodeEnd))
	N += n
	return
}

func (w *Writer) WriteInfo(info string) (N int, err error) {
	n, err := w.writeString(info)
	if err != nil {
		return
	}
	N += n

	n, err = w.WriteCrlf()
	if err != nil {
		return
	}
	N += n

	return
}

func NewWriter(w writer) *Writer {
	return &Writer{writer: w}
}

func NewClientWriter(w writer, continues <-chan bool) *Writer {
	return &Writer{writer: w, continues: continues}
}

type WriterTo interface {
	WriteTo(w *Writer) error
}
