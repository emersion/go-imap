package imap

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// A string that will be quoted.
type Quoted string

// An IMAP writer.
type Writer interface {
	io.Writer

	Flush() error

	writer() *writer
}

type WriterTo interface {
	WriteTo(w Writer) error
}

func formatNumber(num uint32) string {
	return strconv.FormatUint(uint64(num), 10)
}

func FormatDate(t time.Time) string {
	return t.Format("2-Jan-2006")
}

func FormatDateTime(t time.Time) string {
	return t.Format("2-Jan-2006 15:04:05 -0700")
}

// Convert a string list to a field list.
func FormatStringList(list []string) (fields []interface{}) {
	fields = make([]interface{}, len(list))
	for i, v := range list {
		fields[i] = v
	}
	return
}

// Check if a string is 8-bit clean.
func isAscii(s string) bool {
	for _, c := range s {
		if c > unicode.MaxASCII || unicode.IsControl(c) {
			return false
		}
	}
	return true
}

type writer struct {
	io.Writer

	continues <-chan bool
}

func (w *writer) writer() *writer {
	return w
}

// Helper function to write a string to w.
func (w *writer) writeString(s string) error {
	_, err := io.WriteString(w.Writer, s)
	return err
}

func (w *writer) writeCrlf() error {
	if err := w.writeString(crlf); err != nil {
		return err
	}

	return w.Flush()
}

func (w *writer) writeNumber(num uint32) error {
	return w.writeString(formatNumber(num))
}

func (w *writer) writeQuoted(s string) error {
	return w.writeString(strconv.Quote(s))
}

func (w *writer) writeAtom(s string) error {
	return w.writeString(s)
}

func (w *writer) writeAstring(s string) error {
	if !isAscii(s) {
		// IMAP doesn't allow 8-bit data outside literals
		return w.writeLiteral(NewLiteral([]byte(s)))
	}

	specials := string([]rune{dquote, listStart, listEnd, literalStart, sp})
	if strings.ToUpper(s) == nilAtom || s == "" || strings.ContainsAny(s, specials) {
		return w.writeQuoted(s)
	}

	return w.writeAtom(s)
}

func (w *writer) writeDateTime(t time.Time) error {
	if t.IsZero() {
		return w.writeAtom(nilAtom)
	}
	return w.writeQuoted(FormatDateTime(t))
}

func (w *writer) writeFields(fields []interface{}) error {
	for i, field := range fields {
		if i > 0 { // Write separator
			if err := w.writeString(string(sp)); err != nil {
				return err
			}
		}

		if err := w.writeField(field); err != nil {
			return err
		}
	}

	return nil
}

func (w *writer) writeList(fields []interface{}) error {
	if err := w.writeString(string(listStart)); err != nil {
		return err
	}

	if err := w.writeFields(fields); err != nil {
		return err
	}

	return w.writeString(string(listEnd))
}

func (w *writer) writeLiteral(l *Literal) error {
	if l == nil {
		return w.writeString(nilAtom)
	}

	header := string(literalStart) + strconv.Itoa(l.Len()) + string(literalEnd) + crlf
	if err := w.writeString(header); err != nil {
		return err
	}

	// If a channel is available, wait for a continuation request before sending data
	if w.continues != nil {
		// Make sure to flush the writer, otherwise we may never receive a continuation request
		if err := w.Flush(); err != nil {
			return err
		}

		if !<-w.continues {
			return fmt.Errorf("imap: cannot send literal: no continuation request received")
		}
	}

	_, err := w.Write(l.Bytes())
	return err
}

func (w *writer) writeField(field interface{}) error {
	if field == nil {
		return w.writeAtom(nilAtom)
	}

	switch field := field.(type) {
	case string:
		return w.writeAstring(field)
	case Quoted:
		return w.writeQuoted(string(field))
	case int:
		return w.writeNumber(uint32(field))
	case uint32:
		return w.writeNumber(field)
	case *Literal:
		return w.writeLiteral(field)
	case []interface{}:
		return w.writeList(field)
	case time.Time:
		return w.writeDateTime(field)
	case *SeqSet:
		return w.writeString(field.String())
	case *BodySectionName:
		return w.writeString(field.String())
	}

	return fmt.Errorf("imap: cannot format field: %v", field)
}

func (w *writer) writeRespCode(code string, args []interface{}) error {
	if err := w.writeString(string(respCodeStart)); err != nil {
		return err
	}

	fields := []interface{}{code}
	fields = append(fields, args...)

	if err := w.writeFields(fields); err != nil {
		return err
	}

	return w.writeString(string(respCodeEnd))
}

func (w *writer) writeLine(fields ...interface{}) error {
	if err := w.writeFields(fields); err != nil {
		return err
	}

	return w.writeCrlf()
}

func (w *writer) Flush() error {
	f, ok := w.Writer.(interface {
		Flush() error
	})
	if ok {
		return f.Flush()
	}
	return nil
}

func NewWriter(w io.Writer) Writer {
	return &writer{Writer: w}
}

func NewClientWriter(w io.Writer, continues <-chan bool) Writer {
	return &writer{Writer: w, continues: continues}
}
