package imapclient

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func TestReadStatus(t *testing.T) {
	num := func(n uint32) *uint32 { return &n }
	counts := &imap.StatusData{Mailbox: "INBOX", NumMessages: num(11), UIDValidity: 1, NumUnseen: num(7)}
	tests := []struct {
		name, in string
		want     *imap.StatusData
	}{
		{"Empty", `"INBOX" ()`, &imap.StatusData{Mailbox: "INBOX"}},
		{"Items", `"INBOX" (MESSAGES 11 UIDVALIDITY 1 UNSEEN 7)`, counts},
		// imap.139.com sends a SP before the closing paren
		{"TrailingSP", `"INBOX" (MESSAGES 11 UIDVALIDITY 1 UNSEEN 7 )`, counts},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(tc.in+"\r\n")), imapwire.ConnSideClient)
			data, err := readStatus(dec)
			if err != nil {
				t.Fatalf("readStatus() = %v", err)
			}
			if !dec.ExpectCRLF() {
				t.Fatalf("ExpectCRLF() = %v", dec.Err())
			}
			if !reflect.DeepEqual(data, tc.want) {
				t.Errorf("readStatus() = %#v but want %#v", data, tc.want)
			}
		})
	}
}
