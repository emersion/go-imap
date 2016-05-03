package client_test

import (
	"fmt"
	"io"
	"net"
	"testing"

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
