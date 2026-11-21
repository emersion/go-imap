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

func TestExpectListExtraSP(t *testing.T) {
	// Some servers send extra SPs in lists, e.g. "FLAGS (\Seen )" or "LIST ( \HasNoChildren)"
	for _, s := range []string{"( a b)", "(a b )", "(a  b)", "(  a  b  )"} {
		dec := newTestDecoder(s + "\r\n")

		var items []string
		err := dec.ExpectList(func() error {
			var item string
			if !dec.ExpectAtom(&item) {
				return dec.Err()
			}
			items = append(items, item)
			return nil
		})
		if err != nil {
			t.Errorf("ExpectList(%q) = %v", s, err)
		} else if strings.Join(items, ",") != "a,b" || !dec.ExpectCRLF() {
			t.Errorf("ExpectList(%q) = %q, want [a b] then CRLF", s, items)
		}
	}
}

func TestDiscardValueNestedList(t *testing.T) {
	for _, s := range []string{"((a))", "(a (b))", "( (a) )"} {
		dec := newTestDecoder(s + "\r\n")
		if !dec.DiscardValue() || !dec.ExpectCRLF() {
			t.Errorf("DiscardValue(%q) = %v", s, dec.Err())
		}
	}
}
