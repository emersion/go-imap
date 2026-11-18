package imapwire_test

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func newTestDecoder(s string) *imapwire.Decoder {
	return imapwire.NewDecoder(bufio.NewReader(strings.NewReader(s)), imapwire.ConnSideServer)
}

func expectDecoderExpectError(t *testing.T, dec *imapwire.Decoder) {
	t.Helper()
	err := dec.Err()
	if err == nil {
		t.Fatal("decoder has no error")
	}
	var decErr *imapwire.DecoderExpectError
	if !errors.As(err, &decErr) {
		t.Fatalf("decoder error is %T (%v), want *DecoderExpectError", err, err)
	}
}

func TestExpectNumSetInvalid(t *testing.T) {
	dec := newTestDecoder("abc\r\n")

	var numSet imap.NumSet
	if dec.ExpectNumSet(imapwire.NumKindSeq, &numSet) {
		t.Fatal("ExpectNumSet accepted an invalid sequence-set")
	}
	expectDecoderExpectError(t, dec)
}

func TestExpectMailboxInvalidUTF7(t *testing.T) {
	dec := newTestDecoder("&\r\n")

	var name string
	if dec.ExpectMailbox(&name) {
		t.Fatal("ExpectMailbox accepted a name that is not modified UTF-7")
	}
	expectDecoderExpectError(t, dec)
}

func TestExpectMailboxInvalidUTF8(t *testing.T) {
	for _, s := range []string{"\"bad\xffname\"", "\"a\x7fb\"", "{3}\r\na\x01b"} {
		dec := newTestDecoder(s + "\r\n")
		dec.MailboxUTF8 = true

		var name string
		if dec.ExpectMailbox(&name) {
			t.Errorf("ExpectMailbox(%q) accepted an invalid name", s)
			continue
		}
		expectDecoderExpectError(t, dec)
	}
}

func TestExpectMailbox(t *testing.T) {
	tests := []struct {
		s           string
		mailboxUTF8 bool
		want        string
	}{
		{s: "R&-D", want: "R&D"},
		{s: "Caf&AOg-", want: "Cafè"},
		{s: "R&D", mailboxUTF8: true, want: "R&D"},
		{s: `"Cafè"`, mailboxUTF8: true, want: "Cafè"},
		{s: "R&-D", mailboxUTF8: true, want: "R&-D"},
		{s: "inbox", mailboxUTF8: true, want: "INBOX"},
	}
	for _, tc := range tests {
		dec := newTestDecoder(tc.s + "\r\n")
		dec.MailboxUTF8 = tc.mailboxUTF8

		var name string
		if !dec.ExpectMailbox(&name) {
			t.Errorf("ExpectMailbox(%q) = %v", tc.s, dec.Err())
		} else if name != tc.want {
			t.Errorf("ExpectMailbox(%q) = %q, want %q", tc.s, name, tc.want)
		}
	}
}
