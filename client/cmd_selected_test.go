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

func TestClient_Expunge(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		expunged := make(chan uint32, 4)
		err = c.Expunge(expunged)
		if err != nil {
			return
		}

		expected := []uint32{3, 3, 5, 8}

		i := 0
		for id := range expunged {
			if id != expected[i] {
				return fmt.Errorf("Bad expunged sequence number: got %v instead of %v", id, expected[i])
			}
			i++
		}
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "EXPUNGE" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* 3 EXPUNGE\r\n")
		io.WriteString(c, "* 3 EXPUNGE\r\n")
		io.WriteString(c, "* 5 EXPUNGE\r\n")
		io.WriteString(c, "* 8 EXPUNGE\r\n")
		io.WriteString(c, tag + " OK EXPUNGE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Search(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		criteria := []interface{}{"FLAGGED", "SINCE", "1-Feb-1994", "NOT", "FROM", "Smith"}

		results, err := c.Search(criteria)
		if err != nil {
			return
		}

		expected := []uint32{2, 84, 882}
		if fmt.Sprint(results) != fmt.Sprint(expected) {
			return fmt.Errorf("Bad results: %v", results)
		}
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "SEARCH CHARSET UTF-8 FLAGGED SINCE 1-Feb-1994 NOT FROM Smith" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* SEARCH 2 84 882\r\n")
		io.WriteString(c, tag + " OK SEARCH completed\r\n")
	}

	testClient(t, ct, st)
}
