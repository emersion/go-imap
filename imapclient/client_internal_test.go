package imapclient

import (
	"net"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
)

func TestClient_completeCommandStateBeforeDone(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	client := New(clientConn, nil)
	defer client.Close()

	cmd := &loginCommand{}
	cmd.done = make(chan error, 1)

	// With the mutex held, setState blocks, so done must not be signalled yet
	client.mutex.Lock()
	go client.completeCommand(cmd, nil)
	select {
	case <-cmd.done:
		client.mutex.Unlock()
		t.Fatal("done signalled before the state was updated")
	case <-time.After(50 * time.Millisecond):
	}
	client.mutex.Unlock()

	<-cmd.done
	if state := client.State(); state != imap.ConnStateAuthenticated {
		t.Errorf("State() = %v, want %v", state, imap.ConnStateAuthenticated)
	}
}
