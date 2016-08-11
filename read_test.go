package imap_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/emersion/go-imap"
)

func TestParseNumber(t *testing.T) {
	tests := []struct {
		f   interface{}
		n   uint32
		err bool
	}{
		{f: "42", n: 42},
		{f: "0", n: 0},
		{f: "-1", err: true},
		{f: "1.2", err: true},
		{f: nil, err: true},
		{f: imap.NewLiteral([]byte("cc")), err: true},
	}

	for _, test := range tests {
		n, err := imap.ParseNumber(test.f)
		if err != nil {
			if !test.err {
				t.Errorf("Cannot parse number %v", test.f)
			}
		} else {
			if test.err {
				t.Errorf("Parsed invalid number %v", test.f)
			} else if n != test.n {
				t.Errorf("Invalid parsed number: got %v but expected %v", n, test.n)
			}
		}
	}
}

func TestParseStringList(t *testing.T) {
	tests := []struct {
		fields []interface{}
		list   []string
	}{
		{
			fields: []interface{}{"a", "b", "c", "d"},
			list:   []string{"a", "b", "c", "d"},
		},
		{
			fields: []interface{}{"a"},
			list:   []string{"a"},
		},
		{
			fields: []interface{}{},
			list:   []string{},
		},
		{
			fields: []interface{}{"a", 42, "c", "d"},
			list:   nil,
		},
		{
			fields: []interface{}{"a", nil, "c", "d"},
			list:   nil,
		},
	}

	for _, test := range tests {
		list, err := imap.ParseStringList(test.fields)
		if err != nil {
			if test.list != nil {
				t.Errorf("Cannot parse string list: %v", err)
			}
		} else if !reflect.DeepEqual(list, test.list) {
			t.Errorf("Invalid parsed string list: got %v but expected %v", list, test.list)
		}
	}
}

func newReader(s string) (b *bytes.Buffer, r *imap.Reader) {
	b = bytes.NewBuffer([]byte(s))
	r = imap.NewReader(b)
	return
}

func TestReader_ReadSp(t *testing.T) {
	b, r := newReader(" ")
	if err := r.ReadSp(); err != nil {
		t.Error(err)
	}
	if len(b.Bytes()) > 0 {
		t.Error("Buffer is not empty after read")
	}

	b, r = newReader("")
	if err := r.ReadSp(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadCrlf(t *testing.T) {
	b, r := newReader("\r\n")
	if err := r.ReadCrlf(); err != nil {
		t.Error(err)
	}
	if len(b.Bytes()) > 0 {
		t.Error("Buffer is not empty after read")
	}

	b, r = newReader("")
	if err := r.ReadCrlf(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("\r")
	if err := r.ReadCrlf(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("\r42")
	if err := r.ReadCrlf(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadAtom(t *testing.T) {
	b, r := newReader("NIL\r\n")
	if atom, err := r.ReadAtom(); err != nil {
		t.Error(err)
	} else if atom != nil {
		t.Error("NIL atom is not nil:", atom)
	} else {
		if err := r.ReadCrlf(); err != nil && err != io.EOF {
			t.Error("Cannot read CRLF after atom:", err)
		}
		if len(b.Bytes()) > 0 {
			t.Error("Buffer is not empty after read")
		}
	}

	b, r = newReader("atom\r\n")
	if atom, err := r.ReadAtom(); err != nil {
		t.Error(err)
	} else if s, ok := atom.(string); !ok || s != "atom" {
		t.Error("String atom has not the expected value:", atom)
	} else {
		if err := r.ReadCrlf(); err != nil && err != io.EOF {
			t.Error("Cannot read CRLF after atom:", err)
		}
		if len(b.Bytes()) > 0 {
			t.Error("Buffer is not empty after read")
		}
	}

	_, r = newReader("")
	if _, err := r.ReadAtom(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	_, r = newReader("(hi there)\r\n")
	if _, err := r.ReadAtom(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	_, r = newReader("{42}\r\n")
	if _, err := r.ReadAtom(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	_, r = newReader("\"\r\n")
	if _, err := r.ReadAtom(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	_, r = newReader("abc]")
	if _, err := r.ReadAtom(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	_, r = newReader("[abc]def]ghi")
	if _, err := r.ReadAtom(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadLiteral(t *testing.T) {
	b, r := newReader("{7}\r\nabcdefg")
	if literal, err := r.ReadLiteral(); err != nil {
		t.Error(err)
	} else if literal.String() != "abcdefg" {
		t.Error("Literal has not the expected value:", literal.String())
	} else if len(b.Bytes()) > 0 {
		t.Error("Buffer is not empty after read")
	}

	b, r = newReader("")
	if _, err := r.ReadLiteral(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("[7}\r\nabcdefg")
	if _, err := r.ReadLiteral(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("{7]\r\nabcdefg")
	if _, err := r.ReadLiteral(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("{7.4}\r\nabcdefg")
	if _, err := r.ReadLiteral(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("{7}abcdefg")
	if _, err := r.ReadLiteral(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("{7}\rabcdefg")
	if _, err := r.ReadLiteral(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("{7}\nabcdefg")
	if _, err := r.ReadLiteral(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("{7}\r\nabcd")
	if _, err := r.ReadLiteral(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadQuotedString(t *testing.T) {
	b, r := newReader("\"hello gopher\"\r\n")
	if s, err := r.ReadQuotedString(); err != nil {
		t.Error(err)
	} else if s != "hello gopher" {
		t.Error("Quoted string has not the expected value:", s)
	} else {
		if err := r.ReadCrlf(); err != nil && err != io.EOF {
			t.Error("Cannot read CRLF after quoted string:", err)
		}
		if len(b.Bytes()) > 0 {
			t.Error("Buffer is not empty after read")
		}
	}

	b, r = newReader("")
	if _, err := r.ReadQuotedString(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("hello gopher\"\r\n")
	if _, err := r.ReadQuotedString(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("\"hello gopher\r\n")
	if _, err := r.ReadQuotedString(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadFields(t *testing.T) {
	b, r := newReader("field1 \"field2\"\r\n")
	if fields, err := r.ReadFields(); err != nil {
		t.Error(err)
	} else if len(fields) != 2 {
		t.Error("Expected 2 fields, but got", len(fields))
	} else if s, ok := fields[0].(string); !ok || s != "field1" {
		t.Error("Field 1 has not the expected value:", fields[0])
	} else if s, ok := fields[1].(string); !ok || s != "field2" {
		t.Error("Field 2 has not the expected value:", fields[1])
	} else {
		if err := r.ReadCrlf(); err != nil && err != io.EOF {
			t.Error("Cannot read CRLF after fields:", err)
		}
		if len(b.Bytes()) > 0 {
			t.Error("Buffer is not empty after read")
		}
	}

	b, r = newReader("")
	if _, err := r.ReadFields(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("fi\"eld1 \"field2\"\r\n")
	if _, err := r.ReadFields(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("field1 ")
	if _, err := r.ReadFields(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("field1 (")
	if _, err := r.ReadFields(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("field1\"field2\"\r\n")
	if _, err := r.ReadFields(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("\"field1\"\"field2\"\r\n")
	if _, err := r.ReadFields(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadList(t *testing.T) {
	b, r := newReader("(field1 \"field2\" {6}\r\nfield3 field4)")
	if fields, err := r.ReadList(); err != nil {
		t.Error(err)
	} else if len(fields) != 4 {
		t.Error("Expected 2 fields, but got", len(fields))
	} else if s, ok := fields[0].(string); !ok || s != "field1" {
		t.Error("Field 1 has not the expected value:", fields[0])
	} else if s, ok := fields[1].(string); !ok || s != "field2" {
		t.Error("Field 2 has not the expected value:", fields[1])
	} else if literal, ok := fields[2].(*imap.Literal); !ok || literal.String() != "field3" {
		t.Error("Field 3 has not the expected value:", fields[2])
	} else if s, ok := fields[3].(string); !ok || s != "field4" {
		t.Error("Field 4 has not the expected value:", fields[3])
	} else if len(b.Bytes()) > 0 {
		t.Error("Buffer is not empty after read")
	}

	b, r = newReader("()")
	if fields, err := r.ReadList(); err != nil {
		t.Error(err)
	} else if len(fields) != 0 {
		t.Error("Expected 0 fields, but got", len(fields))
	} else if len(b.Bytes()) > 0 {
		t.Error("Buffer is not empty after read")
	}

	b, r = newReader("")
	if _, err := r.ReadList(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("[field1 field2 field3)")
	if _, err := r.ReadList(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("(field1 fie\"ld2 field3)")
	if _, err := r.ReadList(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("(field1 field2 field3\r\n")
	if _, err := r.ReadList(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadLine(t *testing.T) {
	b, r := newReader("field1 field2\r\n")
	if fields, err := r.ReadLine(); err != nil {
		t.Error(err)
	} else if len(fields) != 2 {
		t.Error("Expected 2 fields, but got", len(fields))
	} else if s, ok := fields[0].(string); !ok || s != "field1" {
		t.Error("Field 1 has not the expected value:", fields[0])
	} else if s, ok := fields[1].(string); !ok || s != "field2" {
		t.Error("Field 2 has not the expected value:", fields[1])
	} else if len(b.Bytes()) > 0 {
		t.Error("Buffer is not empty after read")
	}

	b, r = newReader("")
	if _, err := r.ReadLine(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("field1 field2\rabc")
	if _, err := r.ReadLine(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadRespCode(t *testing.T) {
	b, r := newReader("[CAPABILITY NOOP STARTTLS]")
	if code, fields, err := r.ReadRespCode(); err != nil {
		t.Error(err)
	} else if code != "CAPABILITY" {
		t.Error("Response code has not the expected value:", code)
	} else if len(fields) != 2 {
		t.Error("Expected 2 fields, but got", len(fields))
	} else if s, ok := fields[0].(string); !ok || s != "NOOP" {
		t.Error("Field 1 has not the expected value:", fields[0])
	} else if s, ok := fields[1].(string); !ok || s != "STARTTLS" {
		t.Error("Field 2 has not the expected value:", fields[1])
	} else if len(b.Bytes()) > 0 {
		t.Error("Buffer is not empty after read")
	}

	b, r = newReader("")
	if _, _, err := r.ReadRespCode(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("{CAPABILITY NOOP STARTTLS]")
	if _, _, err := r.ReadRespCode(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("[CAPABILITY NO\"OP STARTTLS]")
	if _, _, err := r.ReadRespCode(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("[]")
	if _, _, err := r.ReadRespCode(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("[{3}\r\nabc]")
	if _, _, err := r.ReadRespCode(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("[CAPABILITY NOOP STARTTLS\r\n")
	if _, _, err := r.ReadRespCode(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}

func TestReader_ReadInfo(t *testing.T) {
	b, r := newReader("I love potatoes.\r\n")
	if info, err := r.ReadInfo(); err != nil {
		t.Error(err)
	} else if info != "I love potatoes." {
		t.Error("Info has not the expected value:", info)
	} else if len(b.Bytes()) > 0 {
		t.Error("Buffer is not empty after read")
	}

	b, r = newReader("I love potatoes.")
	if _, err := r.ReadInfo(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("I love potatoes.\r")
	if _, err := r.ReadInfo(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("I love potatoes.\n")
	if _, err := r.ReadInfo(); err == nil {
		t.Error("Invalid read didn't fail")
	}

	b, r = newReader("I love potatoes.\rabc")
	if _, err := r.ReadInfo(); err == nil {
		t.Error("Invalid read didn't fail")
	}
}
