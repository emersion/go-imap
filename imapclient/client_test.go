package imapclient_test

import (
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

const (
	testUsername = "test-user"
	testPassword = "test-password"
)

const simpleRawMessage = `MIME-Version: 1.0
Message-Id: <191101702316132@example.com>
Content-Transfer-Encoding: 8bit
Content-Type: text/plain; charset=utf-8

This is my letter!`

func newClientServerPair(t *testing.T, initialState imap.ConnState) (*imapclient.Client, io.Closer) {
	memServer := imapmemserver.New()

	user := imapmemserver.NewUser(testUsername, testPassword)
	user.Create("INBOX", nil)

	_, err := user.Append("INBOX", strings.NewReader(simpleRawMessage), &imap.AppendOptions{})
	if err != nil {
		t.Fatalf("Append() = %v", err)
	}

	memServer.AddUser(user)

	server := imapserver.New(&imapserver.Options{
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memServer.NewSession(), nil, nil
		},
		InsecureAuth: true,
	})

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}

	go func() {
		if err := server.Serve(ln); err != nil {
			t.Errorf("Serve() = %v", err)
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial() = %v", err)
	}

	client := imapclient.New(conn, nil)

	if initialState >= imap.ConnStateAuthenticated {
		if err := client.Login(testUsername, testPassword).Wait(); err != nil {
			t.Fatalf("Login().Wait() = %v", err)
		}
	}
	if initialState >= imap.ConnStateSelected {
		if _, err := client.Select("INBOX", nil).Wait(); err != nil {
			t.Fatalf("Select().Wait() = %v", err)
		}
	}

	return client, server
}

func TestLogin(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateNotAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Errorf("Login().Wait() = %v", err)
	}
}

func TestLogout(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer server.Close()

	if err := client.Logout().Wait(); err != nil {
		t.Errorf("Logout().Wait() = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Errorf("Close() = %v", err)
	}
}

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

// https://github.com/emersion/go-imap/issues/562
func TestFetch_invalid(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	_, err := client.Fetch(imap.UIDSet(nil), nil).Collect()
	if err == nil {
		t.Fatalf("UIDFetch().Collect() = %v", err)
	}
}

func TestFetch_closeUnreadBody(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	fetchCmd := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{
				Specifier: imap.PartSpecifierNone,
				Peek:      true,
			},
		},
	})
	if err := fetchCmd.Close(); err != nil {
		t.Fatalf("UIDFetch().Close() = %v", err)
	}
}

func TestWaitGreeting_eof(t *testing.T) {
	// bad server: connected but without greeting
	clientConn, serverConn := net.Pipe()

	client := imapclient.New(clientConn, nil)
	defer client.Close()

	if err := serverConn.Close(); err != nil {
		t.Fatalf("serverConn.Close() = %v", err)
	}

	if err := client.WaitGreeting(); err == nil {
		t.Fatalf("WaitGreeting() should fail")
	}
}
