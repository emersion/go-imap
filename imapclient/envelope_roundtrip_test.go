package imapclient_test

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap/v2"
)

// TestFetchEnvelopeUTF8RoundTrip appends a message with UTF-8 in From/To
// headers, then FETCHes ENVELOPE under UTF8=ACCEPT and checks the
// address fields come back intact. This exercises the server side
// (ExtractEnvelope + writeEnvelope) and the client side (readEnvelope)
// across the wire in UTF-8 mode (RFC 9755 §3).
func TestFetchEnvelopeUTF8RoundTrip(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapUTF8Accept) {
		t.Skipf("missing UTF8=ACCEPT support")
	}
	if _, err := client.Enable(imap.CapUTF8Accept).Wait(); err != nil {
		t.Fatalf("Enable(CapUTF8Accept) = %v", err)
	}

	const rawMessage = "MIME-Version: 1.0\r\n" +
		"Message-Id: <utf8-roundtrip@grå.org>\r\n" +
		"From: Grå <grå@grå.org>\r\n" +
		"To: 阿Q <阿Q@例子.中国>, Gøril <gøril@example.com>\r\n" +
		"Subject: hilsen frå Grå\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n" +
		"\r\n" +
		"hei\r\n"

	appendCmd := client.Append("INBOX", int64(len(rawMessage)), nil)
	appendCmd.Write([]byte(rawMessage))
	appendCmd.Close()
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("Append.Wait() = %v", err)
	}

	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select.Wait() = %v", err)
	}

	// newClientServerPair appends a simple ASCII message first, so ours
	// is sequence number 2.
	messages, err := client.Fetch(imap.SeqSetNum(2), &imap.FetchOptions{Envelope: true}).Collect()
	if err != nil {
		t.Fatalf("Fetch = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
	env := messages[0].Envelope
	if env == nil {
		t.Fatalf("Envelope is nil")
	}

	wantFrom := []imap.Address{{Name: "Grå", Mailbox: "grå", Host: "grå.org"}}
	wantTo := []imap.Address{
		{Name: "阿Q", Mailbox: "阿Q", Host: "例子.中国"},
		{Name: "Gøril", Mailbox: "gøril", Host: "example.com"},
	}
	if !reflect.DeepEqual(env.From, wantFrom) {
		t.Errorf("From = %#v, want %#v", env.From, wantFrom)
	}
	if !reflect.DeepEqual(env.To, wantTo) {
		t.Errorf("To = %#v, want %#v", env.To, wantTo)
	}
	if env.Subject != "hilsen frå Grå" {
		t.Errorf("Subject = %q, want %q", env.Subject, "hilsen frå Grå")
	}
}
