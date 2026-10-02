package imapclient_test

import (
	"bytes"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestAppend(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	body := "This is a test message."

	appendCmd := client.Append("INBOX", int64(len(body)), nil)
	if _, err := appendCmd.Write([]byte(body)); err != nil {
		t.Fatalf("AppendCommand.Write() = %v", err)
	}
	if err := appendCmd.Close(); err != nil {
		t.Fatalf("AppendCommand.Close() = %v", err)
	}
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("AppendCommand.Wait() = %v", err)
	}

	// TODO: fetch back message and check body
}

func TestAppendBinary(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// A body with a NUL byte can only be transmitted safely as a literal8.
	body := []byte("Subject: bin\r\n\r\nbefore\x00after")

	appendCmd := client.Append("INBOX", int64(len(body)), &imap.AppendOptions{Binary: true})
	if _, err := appendCmd.Write(body); err != nil {
		t.Fatalf("AppendCommand.Write() = %v", err)
	}
	if err := appendCmd.Close(); err != nil {
		t.Fatalf("AppendCommand.Close() = %v", err)
	}
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("AppendCommand.Wait() = %v", err)
	}

	// The setup appends one message before this one, so ours is the second.
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select().Wait() = %v", err)
	}
	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}
	messages, err := client.Fetch(imap.SeqSetNum(2), fetchOptions).Collect()
	if err != nil {
		t.Fatalf("Fetch() = %v", err)
	} else if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}

	b := messages[0].FindBodySection(bodySection)
	if b == nil {
		t.Fatalf("FindBodySection() = nil")
	}
	if !bytes.Equal(b, body) {
		t.Errorf("body mismatch: got %q, want %q", b, body)
	}
}
