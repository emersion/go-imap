package imapserver_test

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

const (
	idleTestUsername = "shinji"
	idleTestPassword = "eva01"
)

// startIdleTestServer runs an in-memory server with a single user and an INBOX.
func startIdleTestServer(t *testing.T) (host string, port int) {
	t.Helper()

	user := imapmemserver.NewUser(idleTestUsername, idleTestPassword)
	if err := user.Create("INBOX", nil); err != nil {
		t.Fatalf("create INBOX: %v", err)
	}

	mem := imapmemserver.New()
	mem.AddUser(user)

	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		InsecureAuth: true,
	})
	t.Cleanup(func() { _ = srv.Close() })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = srv.Serve(ln) }()

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port
}

func dialIdleTestClient(t *testing.T, host string, port int, opts *imapclient.Options) *imapclient.Client {
	t.Helper()

	c, err := imapclient.DialInsecure(fmt.Sprintf("%s:%d", host, port), opts)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Login(idleTestUsername, idleTestPassword).Wait(); err != nil {
		t.Fatalf("login: %v", err)
	}
	return c
}

func appendIdleTestMessage(t *testing.T, host string, port int, subject string) {
	t.Helper()

	c := dialIdleTestClient(t, host, port, nil)
	body := strings.Join([]string{
		"From: misato@example.com",
		"To: shinji@example.com",
		"Subject: " + subject,
		"Date: " + time.Now().Format(time.RFC1123Z),
		"",
		"Get in the robot.",
		"",
	}, "\r\n")

	cmd := c.Append("INBOX", int64(len(body)), nil)
	if _, err := cmd.Write([]byte(body)); err != nil {
		t.Fatalf("write APPEND literal: %v", err)
	}
	if err := cmd.Close(); err != nil {
		t.Fatalf("close APPEND literal: %v", err)
	}
	if _, err := cmd.Wait(); err != nil {
		t.Fatalf("APPEND: %v", err)
	}
}

// An update queued while nothing is listening must still be announced by the
// next IDLE.
//
// SessionTracker.queueUpdate always appends to the queue, but only signals a
// listener that already exists, and t.updates is nil whenever Idle is not
// running. So a message that arrives while the client is between DONE and its
// next IDLE is queued and then never announced: Idle's loop only polls when
// signalled, so it waits for the *next* update to carry the pending one, and on
// a quiet mailbox that never comes.
//
// The same hole is reachable at startup, because handleIdle writes the
// "+ idling" continuation before starting the goroutine that calls Idle — a
// client is told it is idling before anything is listening. That form is a race
// and awkward to test; this one is deterministic and has the same cause.
func TestIdleAnnouncesUpdatesQueuedWhileNotIdling(t *testing.T) {
	host, port := startIdleTestServer(t)

	exists := make(chan uint32, 8)
	c := dialIdleTestClient(t, host, port, &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data != nil && data.NumMessages != nil {
					select {
					case exists <- *data.NumMessages:
					default:
					}
				}
			},
		},
	})

	if _, err := c.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("SELECT: %v", err)
	}

	// Idle and stop again, the way a client does when it needs the connection
	// to run a command. The server's tracker now has no listener.
	idle, err := c.Idle()
	if err != nil {
		t.Fatalf("IDLE: %v", err)
	}
	if err := idle.Close(); err != nil {
		t.Fatalf("DONE: %v", err)
	}
	drainExists(exists)

	// Delivered by somebody else while this client is not idling.
	appendIdleTestMessage(t, host, port, "Angel attack")

	idle, err = c.Idle()
	if err != nil {
		t.Fatalf("second IDLE: %v", err)
	}
	defer idle.Close()

	select {
	case n := <-exists:
		if n == 0 {
			t.Errorf("EXISTS reported %d messages", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the message delivered between DONE and IDLE was never announced")
	}
}

func drainExists(ch <-chan uint32) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
