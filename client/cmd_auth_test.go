package client_test

import (
	"io"
	"net"
	"testing"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/client"
)

func TestClient_Select(t *testing.T) {
	ct := func(c *client.Client) {
		c.State = common.AuthenticatedState

		mbox, err := c.Select("INBOX", false)
		if err != nil {
			t.Fatal(err)
		}

		if mbox.Name != "INBOX" {
			t.Fatal("Bad mailbox name:", mbox.Name)
		}
		if mbox.ReadOnly {
			t.Fatal("Bad mailbox read-only:", mbox.ReadOnly)
		}
		if len(mbox.Flags) != 5 {
			t.Fatal("Bad mailbox flags:", mbox.Flags)
		}
		if len(mbox.PermanentFlags) != 3 {
			t.Fatal("Bad mailbox permanent flags:", mbox.PermanentFlags)
		}
		if mbox.Messages != 172 {
			t.Fatal("Bad mailbox messages:", mbox.Messages)
		}
		if mbox.Recent != 1 {
			t.Fatal("Bad mailbox recent:", mbox.Recent)
		}
		if mbox.Unseen != 12 {
			t.Fatal("Bad mailbox unseen:", mbox.Unseen)
		}
		if mbox.UidNext != 4392 {
			t.Fatal("Bad mailbox UIDNEXT:", mbox.UidNext)
		}
		if mbox.UidValidity != 3857529045 {
			t.Fatal("Bad mailbox UIDVALIDITY:", mbox.UidValidity)
		}
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
