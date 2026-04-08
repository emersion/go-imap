package imapclient_test

import (
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
)

// TestClient_Closed tests that the Closed() channel is closed when the
// connection is explicitly closed via Close().
func TestClient_Closed(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer server.Close()

	closedCh := client.Closed()
	if closedCh == nil {
		t.Fatal("Closed() returned nil channel")
	}

	select {
	case <-closedCh:
		t.Fatal("Closed() channel closed before calling Close()")
	default: // Expected
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}

	select {
	case <-closedCh:
		t.Log("Closed() channel properly closed after Close()")
	case <-time.After(2 * time.Second):
		t.Fatal("Closed() channel not closed after Close()")
	}
}
