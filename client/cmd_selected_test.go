package client_test

import (
	"io"
	"fmt"
	"net"
	"testing"

	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/client"
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

		date, _ := common.ParseSearchDate("1-Feb-1994")
		criteria := &common.SearchCriteria{
			Deleted: true,
			From: "Smith",
			Since: date,
			Not: &common.SearchCriteria{To: "Pauline"},
		}

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
		if cmd != "SEARCH CHARSET UTF-8 DELETED FROM Smith NOT (TO Pauline) SINCE 1-Feb-1994" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* SEARCH 2 84 882\r\n")
		io.WriteString(c, tag + " OK SEARCH completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Search_Uid(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		criteria := &common.SearchCriteria{
			Undeleted: true,
		}

		results, err := c.UidSearch(criteria)
		if err != nil {
			return
		}

		expected := []uint32{1, 78, 2010}
		if fmt.Sprint(results) != fmt.Sprint(expected) {
			return fmt.Errorf("Bad results: %v", results)
		}
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "UID SEARCH CHARSET UTF-8 UNDELETED" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* SEARCH 1 78 2010\r\n")
		io.WriteString(c, tag + " OK UID SEARCH completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Fetch(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		seqset, _ := common.NewSeqSet("2:3")
		fields := []string{"UID", "BODY[]"}
		messages := make(chan *common.Message, 2)

		err = c.Fetch(seqset, fields, messages)
		if err != nil {
			return
		}

		msg := <-messages
		body := msg.GetBody("BODY[]").String()
		if msg.SeqNum != 2 {
			return fmt.Errorf("First message has bad sequence number: %v", msg.SeqNum)
		}
		if msg.Uid != 42 {
			return fmt.Errorf("First message has bad UID: %v", msg.Uid)
		}
		if body != "I love potatoes." {
			return fmt.Errorf("First message has bad body: %v", body)
		}

		msg = <-messages
		body = msg.GetBody("BODY[]").String()
		if msg.SeqNum != 3 {
			return fmt.Errorf("First message has bad sequence number: %v", msg.SeqNum)
		}
		if msg.Uid != 28 {
			return fmt.Errorf("Second message has bad UID: %v", msg.Uid)
		}
		if body != "Hello World!" {
			return fmt.Errorf("Second message has bad body: %v", body)
		}

		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "FETCH 2:3 (UID BODY[])" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* 2 FETCH (UID 42 BODY[] {16}\r\n")
		io.WriteString(c, "I love potatoes.")
		io.WriteString(c, ")\r\n")

		io.WriteString(c, "* 3 FETCH (UID 28 BODY[] {12}\r\n")
		io.WriteString(c, "Hello World!")
		io.WriteString(c, ")\r\n")

		io.WriteString(c, tag + " OK FETCH completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Fetch_Uid(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		seqset, _ := common.NewSeqSet("1:867")
		fields := []string{"FLAGS"}
		messages := make(chan *common.Message, 1)

		err = c.UidFetch(seqset, fields, messages)
		if err != nil {
			return
		}

		msg := <-messages
		if msg.SeqNum != 23 {
			return fmt.Errorf("First message has bad sequence number: %v", msg.SeqNum)
		}
		if msg.Uid != 42 {
			return fmt.Errorf("Message has bad UID: %v", msg.Uid)
		}
		if len(msg.Flags) != 1 || msg.Flags[0] != "\\Seen" {
			return fmt.Errorf("Message has bad flags: %v", msg.Flags)
		}

		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "UID FETCH 1:867 (FLAGS)" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* 23 FETCH (UID 42 FLAGS (\\Seen))\r\n")

		io.WriteString(c, tag + " OK UID FETCH completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Store(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		updates := make(chan *common.Message, 1)
		seqset, _ := common.NewSeqSet("2")
		err = c.Store(seqset, common.AddFlags, []interface{}{"\\Seen"}, updates)
		if err != nil {
			return
		}

		msg := <-updates
		if len(msg.Flags) != 1 || msg.Flags[0] != "\\Seen" {
			return fmt.Errorf("Bad message flags: %v", msg.Flags)
		}

		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "STORE 2 +FLAGS (\\Seen)" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* 2 FETCH (FLAGS (\\Seen))\r\n")
		io.WriteString(c, tag + " OK STORE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Store_Silent(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		seqset, _ := common.NewSeqSet("2:3")
		err = c.Store(seqset, common.AddFlags, []interface{}{"\\Seen"}, nil)
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "STORE 2:3 +FLAGS.SILENT (\\Seen)" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " OK STORE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Store_Uid(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		seqset, _ := common.NewSeqSet("27:901")
		err = c.UidStore(seqset, common.AddFlags, []interface{}{"\\Deleted"}, nil)
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "UID STORE 27:901 +FLAGS.SILENT (\\Deleted)" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " OK UID STORE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Copy(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		seqset, _ := common.NewSeqSet("2:4")
		err = c.Copy(seqset, "Sent")
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "COPY 2:4 Sent" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " OK COPY completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Copy_Uid(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		seqset, _ := common.NewSeqSet("78:102")
		err = c.UidCopy(seqset, "Drafts")
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "UID COPY 78:102 Drafts" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " OK UID COPY completed\r\n")
	}

	testClient(t, ct, st)
}
