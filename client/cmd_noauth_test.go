package client_test

import (
	"io"
	"fmt"
	"net"
	"testing"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/client"
)

func TestClient_Login_Success(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		err = c.Login("username", "password")
		if err != nil {
			return
		}

		if c.State != common.AuthenticatedState {
			return fmt.Errorf("Client is not in authenticated state after login")
		}

		return
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
	ct := func(c *client.Client) error {
		err := c.Login("username", "password")
		if err == nil {
			return fmt.Errorf("Failed login didn't returned an error: %v", err)
		}

		if c.State != common.NotAuthenticatedState {
			return fmt.Errorf("Client state must be NotAuthenticated after failed login, but is: %v", c.State)
		}

		return nil
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
