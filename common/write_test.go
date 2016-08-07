package common

import (
	"bytes"
	"testing"
	"time"
)

func newWriter() (w *writer, b *bytes.Buffer) {
	b = &bytes.Buffer{}
	w = NewWriter(b).writer()
	return
}

func TestWriter_WriteCrlf(t *testing.T) {
	w, b := newWriter()

	if err := w.writeCrlf(); err != nil {
		t.Error(err)
	}
	if b.String() != "\r\n" {
		t.Error("Not a CRLF")
	}
}

func TestWriter_WriteField_Nil(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField(nil); err != nil {
		t.Error(err)
	}
	if b.String() != "NIL" {
		t.Error("Not NIL")
	}
}

func TestWriter_WriteField_Number(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField(uint32(42)); err != nil {
		t.Error(err)
	}
	if b.String() != "42" {
		t.Error("Not the expected number")
	}
}

func TestWriter_WriteField_Atom(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField("BODY[]"); err != nil {
		t.Error(err)
	}
	if b.String() != "BODY[]" {
		t.Error("Not the expected atom")
	}
}

func TestWriter_WriteString_Quoted(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField("I love potatoes!"); err != nil {
		t.Error(err)
	}
	if b.String() != "\"I love potatoes!\"" {
		t.Error("Not the expected quoted string")
	}
}

func TestWriter_WriteString_Quoted_WithSpecials(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField("I love \"1984\"!"); err != nil {
		t.Error(err)
	}
	if b.String() != "\"I love \\\"1984\\\"!\"" {
		t.Error("Not the expected quoted string")
	}
}

func TestWriter_WriteField_ForcedQuoted(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField(Quoted("dille")); err != nil {
		t.Error(err)
	}
	if b.String() != "\"dille\"" {
		t.Error("Not the expected atom:", b.String())
	}
}

func TestWriter_WriteField_8bitString(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField("☺"); err != nil {
		t.Error(err)
	}
	if b.String() != "{3}\r\n☺" {
		t.Error("Not the expected atom")
	}
}

func TestWriter_WriteField_NilString(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField("NIL"); err != nil {
		t.Error(err)
	}
	if b.String() != "\"NIL\"" {
		t.Error("Not the expected quoted string")
	}
}

func TestWriter_WriteField_EmptyString(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField(""); err != nil {
		t.Error(err)
	}
	if b.String() != "\"\"" {
		t.Error("Not the expected quoted string")
	}
}

func TestWriter_WriteField_DateTime(t *testing.T) {
	w, b := newWriter()

	dt := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	if err := w.writeField(dt); err != nil {
		t.Error(err)
	}
	if b.String() != "\"10-Nov-2009 23:00:00 +0000\"" {
		t.Error("Invalid date:", b.String())
	}
}

func TestWriter_WriteField_ZeroDateTime(t *testing.T) {
	w, b := newWriter()

	dt := time.Time{}
	if err := w.writeField(dt); err != nil {
		t.Error(err)
	}
	if b.String() != "NIL" {
		t.Error("Invalid nil date:", b.String())
	}
}

func TestWriter_WriteFields(t *testing.T) {
	w, b := newWriter()

	if err := w.writeFields([]interface{}{"hey", 42}); err != nil {
		t.Error(err)
	}
	if b.String() != "hey 42" {
		t.Error("Not the expected fields")
	}
}

func TestWriter_WriteField_SimpleList(t *testing.T) {
	w, b := newWriter()

	if err := w.writeField([]interface{}{"hey", 42}); err != nil {
		t.Error(err)
	}
	if b.String() != "(hey 42)" {
		t.Error("Not the expected list")
	}
}

func TestWriter_WriteField_NestedList(t *testing.T) {
	w, b := newWriter()

	list := []interface{}{
		"toplevel",
		[]interface{}{
			"nested",
			0,
		},
		22,
	}

	if err := w.writeField(list); err != nil {
		t.Error(err)
	}
	if b.String() != "(toplevel (nested 0) 22)" {
		t.Error("Not the expected list")
	}
}

func TestWriter_WriteField_Literal(t *testing.T) {
	w, b := newWriter()

	literal := NewLiteral([]byte("hello world"))

	if err := w.writeField(literal); err != nil {
		t.Error(err)
	}
	if b.String() != "{11}\r\nhello world" {
		t.Error("Not the expected literal")
	}
}

func TestWriter_WriteRespCode_NoArgs(t *testing.T) {
	w, b := newWriter()

	if err := w.writeRespCode("READ-ONLY", nil); err != nil {
		t.Error(err)
	}
	if b.String() != "[READ-ONLY]" {
		t.Error("Not the expected response code")
	}
}

func TestWriter_WriteRespCode_WithArgs(t *testing.T) {
	w, b := newWriter()

	args := []interface{}{"IMAP4rev1", "STARTTLS", "LOGINDISABLED"}
	if err := w.writeRespCode("CAPABILITY", args); err != nil {
		t.Error(err)
	}
	if b.String() != "[CAPABILITY IMAP4rev1 STARTTLS LOGINDISABLED]" {
		t.Error("Not the expected response code")
	}
}

func TestWriter_WriteLine(t *testing.T) {
	w, b := newWriter()

	if err := w.writeLine("*", "OK"); err != nil {
		t.Error(err)
	}
	if b.String() != "* OK\r\n" {
		t.Error("Not the expected line:", b.String())
	}
}
