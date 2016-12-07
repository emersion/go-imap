package client_test

import (
	"errors"
	"io"
	"net"
	"testing"

	"github.com/emersion/go-imap/client"
)

func TestClient_Capability(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		caps, err := c.Capability()
		if err != nil {
			return
		}

		if !caps["XTEST"] {
			err = errors.New("Client hasn't advertised capability")
		}
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "CAPABILITY" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* CAPABILITY IMAP4rev1 XTEST\r\n")
		io.WriteString(c, tag+" OK CAPABILITY completed.\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Noop(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		err = c.Noop()
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "NOOP" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK NOOP completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Logout(t *testing.T) {
	ct := func(c *client.Client) error {
		return c.Logout()
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "LOGOUT" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* BYE Client asked to close connection.\r\n")
		io.WriteString(c, tag+" OK LOGOUT completed.\r\n")
		//c.Close()
	}

	testClient(t, ct, st)
}
