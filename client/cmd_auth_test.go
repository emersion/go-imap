package client_test

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/client"
)

func TestClient_Select(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.AuthenticatedState

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
		io.WriteString(c, tag + " OK SELECT completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_List(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.AuthenticatedState

		mailboxes := make(chan *common.MailboxInfo, 3)
		err = c.List("", "%", mailboxes)
		if err != nil {
			return
		}

		expected := []struct{
			name string
			flags []string
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

			if fmt.Sprint(mbox.Flags) != fmt.Sprint(expected[i].flags) {
				return fmt.Errorf("Bad mailbox flags: %v", mbox.Flags)
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
		io.WriteString(c, tag + " OK LIST completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Status(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.AuthenticatedState

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
		io.WriteString(c, tag + " OK STATUS completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Append(t *testing.T) {
	msg := "Hello World!\r\nHello Gophers!\r\n"

	ct := func(c *client.Client) (err error) {
		c.State = common.AuthenticatedState

		date := time.Unix(1462292515, 0)
		literal := common.NewLiteral([]byte(msg))
		err = c.Append("INBOX", []string{"\\Seen", "\\Draft"}, &date, literal)
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "APPEND INBOX (\\Seen \\Draft) \"Tue, 3 May 2016 18:21:55 +0200\" {30}" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "+ send literal\r\n")

		b := make([]byte, 30)
		if _, err := c.Read(b); err != nil {
			t.Fatal(err)
		}

		if string(b) != msg {
			t.Fatal("Bad literal:", string(b))
		}

		io.WriteString(c, tag + " OK APPEND completed\r\n")
	}

	testClient(t, ct, st)
}
