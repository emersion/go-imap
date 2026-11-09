package imapclient_test

import (
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

	messages, err := client.Fetch(imap.SeqSet([]imap.SeqRange{{Start: 1, Stop: 1}}), &imap.FetchOptions{UID: true}).Collect()
	if err != nil {
		t.Fatalf("Fetch() = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Fetch() returned %d messages, want 1", len(messages))
	}
}

func TestMultiAppend(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer server.Close()
	capability, _ := client.Capability().Wait()
	if _, ok := capability["MULTIAPPEND"]; !ok {
		t.Skip("Server doesn't support MULTIAPPEND")
	}

	multiAppendCmd := client.MultiAppend("INBOX")
	body := "This is test message"
	for i := 0; i < 3; i++ {
		writer, err := multiAppendCmd.CreateMessage(int64(len(body)), nil)
		if err != nil {
			t.Fatalf("MultiAppendCommand.CreateMessage() = %v", err)
		}
		writer.Write([]byte(body))
	}
	if err := multiAppendCmd.Close(); err != nil {
		t.Fatalf("MultiAppendCommand.Close() = %v", err)
	}
	if _, err := multiAppendCmd.Wait(); err != nil {
		t.Fatalf("MultiAppendCommand.Wait() = %v", err)
	}

	messages, err := client.Fetch(imap.SeqSet([]imap.SeqRange{{Start: 1, Stop: 3}}), &imap.FetchOptions{UID: true}).Collect()
	if err != nil {
		t.Fatalf("Fetch() = %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("Fetch() returned %d messages, want 3", len(messages))
	}
}
