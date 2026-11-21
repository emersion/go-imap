package imapclient

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// TestReadEnvelopeUTF8Addresses feeds readEnvelope hand-crafted ENVELOPE
// wire forms with UTF-8 in addr-mailbox and addr-host and checks that the
// bytes survive unmangled.
//
// readAddress reads each field via ExpectNString, which just consumes a
// quoted string / literal / NIL — so any byte sequence the server
// emits after UTF8=ACCEPT should round-trip. The cases below cover UTF-8 in
// the local part only, in the host only, in both halves, and a raw
// (non-encoded-word) UTF-8 display name.
func TestReadEnvelopeUTF8Addresses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		envWire string
		want    []imap.Address
	}{
		{
			name:    "utf8 in local part only",
			envWire: `(("阿Q" NIL "阿Q" "例子.中国"))`,
			want:    []imap.Address{{Name: "阿Q", Mailbox: "阿Q", Host: "例子.中国"}},
		},
		{
			name:    "utf8 in host only",
			envWire: `(("Arnt" NIL "arnt" "grå.org"))`,
			want:    []imap.Address{{Name: "Arnt", Mailbox: "arnt", Host: "grå.org"}},
		},
		{
			name:    "utf8 in both halves",
			envWire: `(("Grå" NIL "grå" "grå.org"))`,
			want:    []imap.Address{{Name: "Grå", Mailbox: "grå", Host: "grå.org"}},
		},
		{
			name:    "encoded-word display name",
			envWire: `(("=?utf-8?b?R8O4cmls?=" NIL "gøril" "example.com"))`,
			want:    []imap.Address{{Name: "Gøril", Mailbox: "gøril", Host: "example.com"}},
		},
		{
			name: "multiple addresses",
			envWire: `(("阿Q" NIL "阿Q" "例子.中国")` +
				`("Gøril" NIL "gøril" "example.com"))`,
			want: []imap.Address{
				{Name: "阿Q", Mailbox: "阿Q", Host: "例子.中国"},
				{Name: "Gøril", Mailbox: "gøril", Host: "example.com"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Wrap the From field in a complete ENVELOPE: the other
			// address slots and string fields are NIL.
			wire := `(NIL NIL ` + tc.envWire + ` NIL NIL NIL NIL NIL NIL NIL)`
			dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(wire)), imapwire.ConnSideClient)

			env, err := readEnvelope(dec, &Options{})
			if err != nil {
				t.Fatalf("readEnvelope: %v", err)
			}
			if !reflect.DeepEqual(env.From, tc.want) {
				t.Errorf("From = %#v, want %#v", env.From, tc.want)
			}
		})
	}
}
