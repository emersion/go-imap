package client_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func TestClient_Select(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		mbox, err := c.Select("INBOX", false)
		if err != nil {
			return
		}

		if mbox.Name != "INBOX" {
			return fmt.Errorf("Bad mailbox name: %v", mbox.Name)
		}
		if mbox.ReadOnly {
			return fmt.Errorf("Bad mailbox read-only: %v", mbox.ReadOnly)
		}
		if len(mbox.Flags) != 5 {
			return fmt.Errorf("Bad mailbox flags: %v", mbox.Flags)
		}
		if len(mbox.PermanentFlags) != 3 {
			return fmt.Errorf("Bad mailbox permanent flags: %v", mbox.PermanentFlags)
		}
		if mbox.Messages != 172 {
			return fmt.Errorf("Bad mailbox messages: %v", mbox.Messages)
		}
		if mbox.Recent != 1 {
			return fmt.Errorf("Bad mailbox recent: %v", mbox.Recent)
		}
		if mbox.Unseen != 12 {
			return fmt.Errorf("Bad mailbox unseen: %v", mbox.Unseen)
		}
		if mbox.UidNext != 4392 {
			return fmt.Errorf("Bad mailbox UIDNEXT: %v", mbox.UidNext)
		}
		if mbox.UidValidity != 3857529045 {
			return fmt.Errorf("Bad mailbox UIDVALIDITY: %v", mbox.UidValidity)
		}
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "SELECT INBOX" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* 172 EXISTS\r\n")
		io.WriteString(c, "* 1 RECENT\r\n")
		io.WriteString(c, "* OK [UNSEEN 12] Message 12 is first unseen\r\n")
		io.WriteString(c, "* OK [UIDVALIDITY 3857529045] UIDs valid\r\n")
		io.WriteString(c, "* OK [UIDNEXT 4392] Predicted next UID\r\n")
		io.WriteString(c, "* FLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft)\r\n")
		io.WriteString(c, "* OK [PERMANENTFLAGS (\\Deleted \\Seen \\*)] Limited\r\n")
		io.WriteString(c, tag+" OK SELECT completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Select_ReadOnly(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		mbox, err := c.Select("INBOX", true)
		if err != nil {
			return
		}

		if mbox.Name != "INBOX" {
			return fmt.Errorf("Bad mailbox name: %v", mbox.Name)
		}
		if !mbox.ReadOnly {
			return fmt.Errorf("Bad mailbox read-only: %v", mbox.ReadOnly)
		}
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "EXAMINE INBOX" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK [READ-ONLY] EXAMINE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Create(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		err = c.Create("New Mailbox")
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "CREATE \"New Mailbox\"" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK CREATE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Delete(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		err = c.Delete("Old Mailbox")
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "DELETE \"Old Mailbox\"" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK DELETE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Rename(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		err = c.Rename("Old Mailbox", "New Mailbox")
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "RENAME \"Old Mailbox\" \"New Mailbox\"" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK RENAME completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Subscribe(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		err = c.Subscribe("Mailbox")
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "SUBSCRIBE Mailbox" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK SUBSCRIBE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Unsubscribe(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		err = c.Unsubscribe("Mailbox")
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "UNSUBSCRIBE Mailbox" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK UNSUBSCRIBE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_List(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		mailboxes := make(chan *imap.MailboxInfo, 3)
		err = c.List("", "%", mailboxes)
		if err != nil {
			return
		}

		expected := []struct {
			name       string
			attributes []string
		}{
			{"INBOX", []string{"flag1"}},
			{"Drafts", []string{"flag2", "flag3"}},
			{"Sent", nil},
		}

		i := 0
		for mbox := range mailboxes {
			if mbox.Name != expected[i].name {
				return fmt.Errorf("Bad mailbox name: %v", mbox.Name)
			}

			if fmt.Sprint(mbox.Attributes) != fmt.Sprint(expected[i].attributes) {
				return fmt.Errorf("Bad mailbox attributes: %v", mbox.Attributes)
			}

			i++
		}

		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "LIST \"\" %" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* LIST (flag1) \"/\" INBOX\r\n")
		io.WriteString(c, "* LIST (flag2 flag3) \"/\" Drafts\r\n")
		io.WriteString(c, "* LIST () \"/\" Sent\r\n")
		io.WriteString(c, tag+" OK LIST completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Lsub(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		mailboxes := make(chan *imap.MailboxInfo, 1)
		err = c.Lsub("", "%", mailboxes)
		if err != nil {
			return
		}

		mbox := <-mailboxes
		if mbox.Name != "INBOX" {
			return fmt.Errorf("Bad mailbox name: %v", mbox.Name)
		}
		if len(mbox.Attributes) != 0 {
			return fmt.Errorf("Bad mailbox flags: %v", mbox.Attributes)
		}

		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "LSUB \"\" %" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* LSUB () \"/\" INBOX\r\n")
		io.WriteString(c, tag+" OK LIST completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Status(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		mbox, err := c.Status("INBOX", []string{"MESSAGES", "RECENT"})
		if err != nil {
			return
		}

		if mbox.Messages != 42 {
			return fmt.Errorf("Bad mailbox messages: %v", mbox.Messages)
		}
		if mbox.Recent != 1 {
			return fmt.Errorf("Bad mailbox recent: %v", mbox.Recent)
		}

		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "STATUS INBOX (MESSAGES RECENT)" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* STATUS INBOX (MESSAGES 42 RECENT 1)\r\n")
		io.WriteString(c, tag+" OK STATUS completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Append(t *testing.T) {
	msg := "Hello World!\r\nHello Gophers!\r\n"

	ct := func(c *client.Client) (err error) {
		c.State = imap.AuthenticatedState

		date := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
		literal := bytes.NewBufferString(msg)
		err = c.Append("INBOX", []string{"\\Seen", "\\Draft"}, date, literal)
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "APPEND INBOX (\\Seen \\Draft) \"10-Nov-2009 23:00:00 +0000\" {30}" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "+ send literal\r\n")

		b := make([]byte, 30)
		if _, err := io.ReadFull(c, b); err != nil {
			t.Fatal(err)
		}

		if string(b) != msg {
			t.Fatal("Bad literal:", string(b))
		}

		io.WriteString(c, tag+" OK APPEND completed\r\n")
	}

	testClient(t, ct, st)
}
