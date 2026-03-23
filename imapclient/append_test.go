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

	// TODO: fetch back message and check body
}
