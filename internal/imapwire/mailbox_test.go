package imapwire

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

// TestMailboxEncodeDecodeAmpersandDash covers the mailbox name "A&-B" in
// both modified-UTF-7 and raw-UTF-8 modes.
//
// In modified UTF-7 (the default), the literal '&' has to be sent as
// "&-", so "A&-B" is wire form "A&--B". In UTF-8 mode (after ENABLE
// IMAP4rev2 or ENABLE UTF8=ACCEPT, RFC 9755), '&' has no special meaning
// and "A&-B" is sent verbatim.
func TestMailboxEncodeDecodeAmpersandDash(t *testing.T) {
	const name = "A&-B"

	for _, tc := range []struct {
		mode     string
		quoted   bool
		wireBody string // bytes between the surrounding quotes
	}{
		{"modified UTF-7", false, "A&--B"},
		{"UTF-8 (RFC 9755)", true, "A&-B"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			var buf bytes.Buffer
			bw := bufio.NewWriter(&buf)
			enc := NewEncoder(bw, ConnSideClient)
			enc.QuotedUTF8 = tc.quoted
			enc.Mailbox(name)
			if err := bw.Flush(); err != nil {
				t.Fatalf("flush: %v", err)
			}
			got := buf.String()
			want := `"` + tc.wireBody + `"`
			if got != want {
				t.Errorf("encode: got %q, want %q", got, want)
			}

			dec := NewDecoder(bufio.NewReader(strings.NewReader(got)), ConnSideServer)
			dec.QuotedUTF8 = tc.quoted
			var roundtrip string
			if !dec.ExpectMailbox(&roundtrip) {
				t.Fatalf("decode failed: %v", dec.Err())
			}
			if roundtrip != name {
				t.Errorf("decode: got %q, want %q", roundtrip, name)
			}
		})
	}
}
