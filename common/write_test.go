package common_test

import (
	"bytes"
	"testing"

	"github.com/emersion/imap/common"
)

func newWriter() (w *common.Writer, b *bytes.Buffer) {
	b = &bytes.Buffer{}
	w = common.NewWriter(b)
	return
}

func TestWriter_WriteSp(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteSp(); err != nil {
		t.Error(err)
	}
	if b.String() != " " {
		t.Error("Not a separator")
	}
}

func TestWriter_WriteCrlf(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteCrlf(); err != nil {
		t.Error(err)
	}
	if b.String() != "\r\n" {
		t.Error("Not a CRLF")
	}
}

func TestWriter_WriteNil(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteNil(); err != nil {
		t.Error(err)
	}
	if b.String() != "NIL" {
		t.Error("Not NIL")
	}
}

func TestWriter_WriteNumber(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteNumber(42); err != nil {
		t.Error(err)
	}
	if b.String() != "42" {
		t.Error("Not the Expected number")
	}
}

func TestWriter_WriteString_Atom(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteString("BODY[]"); err != nil {
		t.Error(err)
	}
	if b.String() != "BODY[]" {
		t.Error("Not the Expected atom")
	}
}

func TestWriter_WriteString_Quoted(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteString("I love potatoes!"); err != nil {
		t.Error(err)
	}
	if b.String() != "\"I love potatoes!\"" {
		t.Error("Not the Expected quoted string")
	}
}

func TestWriter_WriteString_Nil(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteString("NIL"); err != nil {
		t.Error(err)
	}
	if b.String() != "\"NIL\"" {
		t.Error("Not the Expected quoted string")
	}
}

func TestWriter_WriteFields(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteFields([]interface{}{"hey", 42}); err != nil {
		t.Error(err)
	}
	if b.String() != "hey 42" {
		t.Error("Not the Expected fields")
	}
}

func TestWriter_WriteList_Simple(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteList([]interface{}{"hey", 42}); err != nil {
		t.Error(err)
	}
	if b.String() != "(hey 42)" {
		t.Error("Not the Expected list")
	}
}

func TestWriter_WriteList_Nested(t *testing.T) {
	w, b := newWriter()

	list := []interface{}{
		"toplevel",
		[]interface{}{
			"nested",
			0,
		},
		22,
	}

	if _, err := w.WriteList(list); err != nil {
		t.Error(err)
	}
	if b.String() != "(toplevel (nested 0) 22)" {
		t.Error("Not the Expected list")
	}
}

func TestWriter_WriteLiteral(t *testing.T) {
	w, b := newWriter()

	literal := common.NewLiteral([]byte("hello world"))

	if _, err := w.WriteLiteral(literal); err != nil {
		t.Error(err)
	}
	if b.String() != "{11}\r\nhello world" {
		t.Error("Not the Expected literal")
	}
}
