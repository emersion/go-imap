package imapclient

import (
	"bufio"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func decodeBody(t *testing.T, s string) imap.BodyStructure {
	t.Helper()
	dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(s)), imapwire.ConnSideServer)
	bs, err := readBody(dec, &Options{})
	if err != nil {
		t.Fatalf("readBody(%q): %v", s, err)
	}
	return bs
}

// QQ Mail sends NIL instead of a media type/subtype for some single parts.
// See https://github.com/emersion/go-imap/issues/678
func TestReadBodyNILType(t *testing.T) {
	bs := decodeBody(t, `(NIL NIL ("CHARSET" "GB2312") NIL NIL NIL 7 NIL NIL NIL)`)
	sp, ok := bs.(*imap.BodyStructureSinglePart)
	if !ok {
		t.Fatalf("readBody returned %T, want *imap.BodyStructureSinglePart", bs)
	}
	if sp.Type != "" || sp.Subtype != "" {
		t.Errorf("type/subtype = %q/%q, want empty", sp.Type, sp.Subtype)
	}
	if sp.Size != 7 {
		t.Errorf("size = %d, want 7", sp.Size)
	}
	if got := sp.Params["charset"]; got != "GB2312" {
		t.Errorf("params[charset] = %q, want GB2312", got)
	}
}

func TestReadBodyNILSubtype(t *testing.T) {
	bs := decodeBody(t, `("TEXT" NIL ("CHARSET" "US-ASCII") NIL NIL "7BIT" 12 1 NIL NIL NIL)`)
	sp, ok := bs.(*imap.BodyStructureSinglePart)
	if !ok {
		t.Fatalf("readBody returned %T, want *imap.BodyStructureSinglePart", bs)
	}
	if sp.Type != "TEXT" || sp.Subtype != "" {
		t.Errorf("type/subtype = %q/%q, want TEXT/\"\"", sp.Type, sp.Subtype)
	}
	if sp.Text == nil || sp.Text.NumLines != 1 {
		t.Errorf("text part = %+v, want NumLines 1", sp.Text)
	}
}

func TestReadBodyMultipartUnaffected(t *testing.T) {
	bs := decodeBody(t, `(("TEXT" "PLAIN" ("CHARSET" "US-ASCII") NIL NIL "7BIT" 1152 23)("TEXT" "PLAIN" ("CHARSET" "US-ASCII" "NAME" "cc.diff") "<x@y>" "Compiler diff" "BASE64" 4554 73) "MIXED")`)
	mp, ok := bs.(*imap.BodyStructureMultiPart)
	if !ok {
		t.Fatalf("readBody returned %T, want *imap.BodyStructureMultiPart", bs)
	}
	if mp.Subtype != "MIXED" || len(mp.Children) != 2 {
		t.Errorf("subtype/children = %q/%d, want MIXED/2", mp.Subtype, len(mp.Children))
	}
}
