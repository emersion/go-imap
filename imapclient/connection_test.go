package imapclient_test

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
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

type pipeConn struct {
	io.Reader
	io.Writer
	closer io.Closer
}

func (c pipeConn) Close() error {
	return c.closer.Close()
}

func (c pipeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

func (c pipeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}

func (c pipeConn) SetDeadline(t time.Time) error {
	return nil
}

func (c pipeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c pipeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

var _ net.Conn = pipeConn{}

// TestCommand_Wait_ConnectionFailure tests that Wait() returns an error instead
// of hanging when the network connection drops unexpectedly.
func TestCommand_Wait_ConnectionFailure(t *testing.T) {
	// Create a custom connection pair
	clientR, serverW := io.Pipe()
	serverR, clientW := io.Pipe()

	clientConn := pipeConn{
		Reader: clientR,
		Writer: clientW,
		closer: clientW,
	}
	serverConn := pipeConn{
		Reader: serverR,
		Writer: serverW,
		closer: serverW,
	}

	client := imapclient.New(clientConn, nil)
	defer client.Close()

	// Hacky server which sends greeting then closes without responding to commands.
	go func() {
		serverW.Write([]byte("* OK IMAP server ready\r\n"))

		buf := make([]byte, 1024)
		serverR.Read(buf)

		time.Sleep(50 * time.Millisecond)
		serverConn.Close()
	}()

	if err := client.WaitGreeting(); err != nil {
		t.Fatalf("WaitGreeting() = %v", err)
	}

	noopCmd := client.Noop()

	// Wait should return an error, not hang
	errCh := make(chan error, 1)
	go func() {
		errCh <- noopCmd.Wait()
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("Expected error after connection failure, got nil")
		} else {
			t.Logf("Wait() returned error as expected: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() hung after connection failure")
	}
}

// TestMultipleCommands_ConnectionFailure tests that multiple pending commands
// are properly unblocked when the connection drops.
func TestMultipleCommands_ConnectionFailure(t *testing.T) {
	// Create a custom connection pair
	clientR, serverW := io.Pipe()
	serverR, clientW := io.Pipe()

	clientConn := pipeConn{
		Reader: clientR,
		Writer: clientW,
		closer: clientW,
	}
	serverConn := pipeConn{
		Reader: serverR,
		Writer: serverW,
		closer: serverW,
	}

	client := imapclient.New(clientConn, nil)
	defer client.Close()

	// Hacky server which send greeting then closes without responding.
	go func() {
		serverW.Write([]byte("* OK IMAP server ready\r\n"))

		buf := make([]byte, 4096)
		serverR.Read(buf)

		time.Sleep(100 * time.Millisecond)
		serverConn.Close()
	}()

	if err := client.WaitGreeting(); err != nil {
		t.Fatalf("WaitGreeting() = %v", err)
	}

	cmd1 := client.Noop()
	cmd2 := client.Noop()
	cmd3 := client.Noop()

	done := make(chan struct{})
	go func() {
		cmd1.Wait()
		cmd2.Wait()
		cmd3.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("All commands completed after connection failure")
	case <-time.After(5 * time.Second):
		t.Fatal("Commands hung after connection failure")
	}
}
