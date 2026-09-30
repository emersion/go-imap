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
