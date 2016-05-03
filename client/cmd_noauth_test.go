package client_test

import (
	"io"
	"net"
	"testing"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/client"
)

func TestClient_Login_Success(t *testing.T) {
	ct := func(c *client.Client) {
		err := c.Login("username", "password")
		if err != nil {
			t.Fatal(err)
		}

		if c.State != common.AuthenticatedState {
			t.Fatal("Client is not in authenticated state after login")
		}
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "LOGIN username password" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " OK LOGIN completed.\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Login_Error(t *testing.T) {
	ct := func(c *client.Client) {
		err := c.Login("username", "password")
		if err == nil {
			t.Fatal("Failed login didn't returned an error")
		}

		if c.State != common.NotAuthenticatedState {
			t.Fatal("Client state must be NotAuthenticated after failed login")
		}
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "LOGIN username password" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " NO LOGIN incorrect.\r\n")
	}

	testClient(t, ct, st)
}
