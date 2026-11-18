package imapwire_test

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func TestEncoderMailbox(t *testing.T) {
	tests := []struct {
		name                    string
		mailboxUTF8, quotedUTF8 bool
		want                    string
	}{
		{name: "R&D", want: `"R&-D"`},
		{name: "Cafè", quotedUTF8: true, want: `"Caf&AOg-"`},
		{name: "R&D", mailboxUTF8: true, quotedUTF8: true, want: `"R&D"`},
		{name: "Cafè", mailboxUTF8: true, quotedUTF8: true, want: `"Cafè"`},
		{name: "Cafè", mailboxUTF8: true, want: "{5}\r\nCafè"},
		{name: "inbox", mailboxUTF8: true, quotedUTF8: true, want: "INBOX"},
	}
	for _, tc := range tests {
		var buf bytes.Buffer
		enc := imapwire.NewEncoder(bufio.NewWriter(&buf), imapwire.ConnSideServer)
		enc.MailboxUTF8 = tc.mailboxUTF8
		enc.QuotedUTF8 = tc.quotedUTF8

		if err := enc.Mailbox(tc.name).CRLF(); err != nil {
			t.Errorf("Mailbox(%q) = %v", tc.name, err)
		} else if got := buf.String(); got != tc.want+"\r\n" {
			t.Errorf("Mailbox(%q) = %q, want %q", tc.name, got, tc.want+"\r\n")
		}
	}
}
