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
		t.Error("Not the expected number")
	}
}

func TestWriter_WriteString_Atom(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteString("BODY[]"); err != nil {
		t.Error(err)
	}
	if b.String() != "BODY[]" {
		t.Error("Not the expected atom")
	}
}

func TestWriter_WriteString_Quoted(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteString("I love potatoes!"); err != nil {
		t.Error(err)
	}
	if b.String() != "\"I love potatoes!\"" {
		t.Error("Not the expected quoted string")
	}
}

func TestWriter_WriteString_Nil(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteString("NIL"); err != nil {
		t.Error(err)
	}
	if b.String() != "\"NIL\"" {
		t.Error("Not the expected quoted string")
	}
}

func TestWriter_WriteString_Empty(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteString(""); err != nil {
		t.Error(err)
	}
	if b.String() != "\"\"" {
		t.Error("Not the expected quoted string")
	}
}

func TestWriter_WriteFields(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteFields([]interface{}{"hey", 42}); err != nil {
		t.Error(err)
	}
	if b.String() != "hey 42" {
		t.Error("Not the expected fields")
	}
}

func TestWriter_WriteList_Simple(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteList([]interface{}{"hey", 42}); err != nil {
		t.Error(err)
	}
	if b.String() != "(hey 42)" {
		t.Error("Not the expected list")
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
		t.Error("Not the expected list")
	}
}

func TestWriter_WriteLiteral(t *testing.T) {
	w, b := newWriter()

	literal := common.NewLiteral([]byte("hello world"))

	if _, err := w.WriteLiteral(literal); err != nil {
		t.Error(err)
	}
	if b.String() != "{11}\r\nhello world" {
		t.Error("Not the expected literal")
	}
}

func TestWriter_WriteRespCode_NoArgs(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteRespCode("READ-ONLY", nil); err != nil {
		t.Error(err)
	}
	if b.String() != "[READ-ONLY]" {
		t.Error("Not the expected response code")
	}
}

func TestWriter_WriteRespCode_WithArgs(t *testing.T) {
	w, b := newWriter()

	args := []interface{}{"IMAP4rev1", "STARTTLS", "LOGINDISABLED"}
	if _, err := w.WriteRespCode("CAPABILITY", args); err != nil {
		t.Error(err)
	}
	if b.String() != "[CAPABILITY IMAP4rev1 STARTTLS LOGINDISABLED]" {
		t.Error("Not the expected response code")
	}
}

func TestWriter_WriteInfo(t *testing.T) {
	w, b := newWriter()

	if _, err := w.WriteInfo("NOOP completed"); err != nil {
		t.Error(err)
	}
	if b.String() != "NOOP completed\r\n" {
		t.Error("Not the expected info")
	}
}
