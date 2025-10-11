package imapserver_test

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

func TestServer_Notify(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with SELECTED mailbox spec
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY SET (SELECTED (MessageNew MessageExpunge))\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifyNone(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY NONE
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY NONE\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifyMultipleItems(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with multiple items and STATUS
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY SET (STATUS) (SELECTED (MessageNew MessageExpunge)) (PERSONAL (MailboxName SubscriptionChange))\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifySubtree(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with SUBTREE
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY SET (SUBTREE (INBOX) (MessageNew))\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifyMailboxList(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with explicit mailbox list
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY SET ((INBOX) (MessageNew MessageExpunge FlagChange))\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifySelectedDelayed(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with SELECTED-DELAYED
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY SET (SELECTED-DELAYED (MessageNew MessageExpunge))\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifyInboxes(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with INBOXES
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY SET (INBOXES (MessageNew))\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifySubscribed(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with SUBSCRIBED
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY SET (SUBSCRIBED (MessageNew MailboxName))\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifyAllEvents(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with all event types
	// Note: The test implementation refuses NOTIFY with NO [NOTIFICATIONOVERFLOW]
	bw.Write([]byte("a1 NOTIFY SET (SELECTED (MessageNew MessageExpunge FlagChange AnnotationChange MailboxName SubscriptionChange MailboxMetadataChange ServerMetadataChange))\r\n"))
	bw.Flush()

	expectNOWithCode(t, scanner, "a1", "NOTIFICATIONOVERFLOW")
}

func TestServer_NotifyNotAuthenticated(t *testing.T) {
	conn, bw, scanner := newTestClient(t, false)
	defer conn.Close()

	// Test NOTIFY before authentication should fail
	bw.Write([]byte("a1 NOTIFY NONE\r\n"))
	bw.Flush()

	expectBAD(t, scanner, "a1")
}

func TestServer_NotifyInvalidSyntax(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with invalid syntax (missing SET/NONE)
	bw.Write([]byte("a1 NOTIFY (SELECTED (MessageNew))\r\n"))
	bw.Flush()

	expectBAD(t, scanner, "a1")
}

func TestServer_NotifySetNoItems(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY SET without any items should fail
	bw.Write([]byte("a1 NOTIFY SET\r\n"))
	bw.Flush()

	expectBAD(t, scanner, "a1")
}

func TestServer_NotifyInvalidEvent(t *testing.T) {
	conn, bw, scanner := newTestClient(t, true)
	defer conn.Close()

	// Test NOTIFY with invalid event name
	bw.Write([]byte("a1 NOTIFY SET (SELECTED (InvalidEvent))\r\n"))
	bw.Flush()

	expectBAD(t, scanner, "a1")
}

// Helper functions

func newTestClient(t *testing.T, authenticate bool) (net.Conn, *bufio.Writer, *bufio.Scanner) {
	memServer := imapmemserver.New()

	user := imapmemserver.NewUser("testuser", "testpass")
	user.Create("INBOX", nil)
	memServer.AddUser(user)

	server := imapserver.New(&imapserver.Options{
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memServer.NewSession(), nil, nil
		},
		InsecureAuth: true,
		Caps: imap.CapSet{
			imap.CapIMAP4rev1: {},
			imap.CapNotify:    {},
		},
	})

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}

	go func() {
		if err := server.Serve(ln); err != nil {
			// Server errors are expected when closing
		}
	}()
	t.Cleanup(func() {
		server.Close()
		ln.Close()
	})

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial() = %v", err)
	}

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	scanner := bufio.NewScanner(reader)

	// Read greeting
	if !scanner.Scan() {
		t.Fatalf("Failed to read greeting: %v", scanner.Err())
	}
	greeting := scanner.Text()
	if !strings.HasPrefix(greeting, "* OK") {
		t.Fatalf("Unexpected greeting: %v", greeting)
	}

	if authenticate {
		// Login
		writer.Write([]byte("a0 LOGIN testuser testpass\r\n"))
		writer.Flush()

		expectOK(t, scanner, "a0")
	}

	return conn, writer, scanner
}

func expectOK(t *testing.T, scanner *bufio.Scanner, tag string) {
	t.Helper()
	if !scanner.Scan() {
		t.Fatalf("Failed to read response: %v", scanner.Err())
	}
	line := scanner.Text()
	expected := tag + " OK"
	if !strings.HasPrefix(line, expected) {
		t.Fatalf("Expected OK response with tag %v, got: %v", tag, line)
	}
}

func expectNOWithCode(t *testing.T, scanner *bufio.Scanner, tag string, code string) {
	t.Helper()
	if !scanner.Scan() {
		t.Fatalf("Failed to read response: %v", scanner.Err())
	}
	line := scanner.Text()
	expected := tag + " NO [" + code + "]"
	if !strings.HasPrefix(line, expected) {
		t.Fatalf("Expected NO [%v] response with tag %v, got: %v", code, tag, line)
	}
}

func expectNO(t *testing.T, scanner *bufio.Scanner, tag string) {
	t.Helper()
	if !scanner.Scan() {
		t.Fatalf("Failed to read response: %v", scanner.Err())
	}
	line := scanner.Text()
	expected := tag + " NO"
	if !strings.HasPrefix(line, expected) {
		t.Fatalf("Expected NO response with tag %v, got: %v", tag, line)
	}
}

func expectBAD(t *testing.T, scanner *bufio.Scanner, tag string) {
	t.Helper()
	if !scanner.Scan() {
		t.Fatalf("Failed to read response: %v", scanner.Err())
	}
	line := scanner.Text()
	expected := tag + " BAD"
	if !strings.HasPrefix(line, expected) {
		t.Fatalf("Expected BAD response with tag %v, got: %v", tag, line)
	}
}
