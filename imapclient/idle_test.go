package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestIdle(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	idleCmd, err := client.Idle()
	if err != nil {
		t.Fatalf("Idle() = %v", err)
	}
	// TODO: test unilateral updates
	if err := idleCmd.Close(); err != nil {
		t.Errorf("Close() = %v", err)
	}
}

func TestIdle_closedConn(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	idleCmd, err := client.Idle()
	if err != nil {
		t.Fatalf("Idle() = %v", err)
	}
	defer idleCmd.Close()

	if err := client.Close(); err != nil {
		t.Fatalf("client.Close() = %v", err)
	}

	if err := idleCmd.Wait(); err == nil {
		t.Errorf("IdleCommand.Wait() = nil, want an error")
	}
}
