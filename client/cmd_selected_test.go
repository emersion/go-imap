package client_test

import (
	"io"
	"fmt"
	"net"
	"testing"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/client"
)

func TestClient_Check(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		err = c.Check()
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "CHECK" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " OK CHECK completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Close(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState
		c.Mailbox = &common.MailboxStatus{Name: "INBOX"}

		err = c.Close()
		if err != nil {
			return
		}

		if c.State != common.AuthenticatedState {
			return fmt.Errorf("Bad client state: %v", c.State)
		}
		if c.Mailbox != nil {
			return fmt.Errorf("Client selected mailbox is not nil: %v", c.Mailbox)
		}
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "CLOSE" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " OK CLOSE completed\r\n")
	}

	testClient(t, ct, st)
}
