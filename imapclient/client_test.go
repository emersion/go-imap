package imapclient_test

import (
	"io"
	"log"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func ExampleClient() {
	c, err := imapclient.DialTLS("mail.example.org:993", nil)
	if err != nil {
		log.Fatalf("failed to dial IMAP server: %v", err)
	}
	defer c.Close()

	if err := c.Login("root", "asdf").Wait(); err != nil {
		log.Fatalf("failed to login: %v", err)
	}

	mailboxes, err := c.List("", "%", nil).Collect()
	if err != nil {
		log.Fatalf("failed to list mailboxes: %v", err)
	}
	log.Printf("Found %v mailboxes", len(mailboxes))
	for _, mbox := range mailboxes {
		log.Printf(" - %v", mbox.Mailbox)
	}

	selectedMbox, err := c.Select("INBOX", nil).Wait()
	if err != nil {
		log.Fatalf("failed to select INBOX: %v", err)
	}
	log.Printf("INBOX contains %v messages", selectedMbox.NumMessages)

	if selectedMbox.NumMessages > 0 {
		seqSet := imap.SeqSetNum(1)
		fetchItems := []imap.FetchItem{imap.FetchItemEnvelope}
		messages, err := c.Fetch(seqSet, fetchItems, nil).Collect()
		if err != nil {
			log.Fatalf("failed to fetch first message in INBOX: %v", err)
		}
		log.Printf("subject of first message in INBOX: %v", messages[0].Envelope.Subject)
	}

	if err := c.Logout().Wait(); err != nil {
		log.Fatalf("failed to logout: %v", err)
	}
}

func ExampleClient_pipelining() {
	var c *imapclient.Client

	uid := uint32(42)
	fetchItems := []imap.FetchItem{imap.FetchItemEnvelope}

	// Login, select and fetch a message in a single roundtrip
	loginCmd := c.Login("root", "root")
	selectCmd := c.Select("INBOX", nil)
	fetchCmd := c.UIDFetch(imap.SeqSetNum(uid), fetchItems, nil)

	if err := loginCmd.Wait(); err != nil {
		log.Fatalf("failed to login: %v", err)
	}
	if _, err := selectCmd.Wait(); err != nil {
		log.Fatalf("failed to select INBOX: %v", err)
	}
	if messages, err := fetchCmd.Collect(); err != nil {
		log.Fatalf("failed to fetch message: %v", err)
	} else {
		log.Printf("Subject: %v", messages[0].Envelope.Subject)
	}
}

func ExampleClient_Append() {
	var c *imapclient.Client

	buf := []byte("From: <root@nsa.gov>\r\n\r\nHi <3")
	size := int64(len(buf))
	appendCmd := c.Append("INBOX", size, nil)
	if _, err := appendCmd.Write(buf); err != nil {
		log.Fatalf("failed to write message: %v", err)
	}
	if err := appendCmd.Close(); err != nil {
		log.Fatalf("failed to close message: %v", err)
	}
	if _, err := appendCmd.Wait(); err != nil {
		log.Fatalf("APPEND command failed: %v", err)
	}
}

func ExampleClient_Status() {
	var c *imapclient.Client

	options := imap.StatusOptions{NumMessages: true}
	if data, err := c.Status("INBOX", &options).Wait(); err != nil {
		log.Fatalf("STATUS command failed: %v", err)
	} else {
		log.Printf("INBOX contains %v messages", *data.NumMessages)
	}
}

func ExampleClient_List_stream() {
	var c *imapclient.Client

	// ReturnStatus requires server support for IMAP4rev2 or LIST-STATUS
	listCmd := c.List("", "%", &imap.ListOptions{
		ReturnStatus: &imap.StatusOptions{
			NumMessages: true,
			NumUnseen:   true,
		},
	})
	for {
		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
		log.Printf("Mailbox %q contains %v messages (%v unseen)", mbox.Mailbox, mbox.Status.NumMessages, mbox.Status.NumUnseen)
	}
	if err := listCmd.Close(); err != nil {
		log.Fatalf("LIST command failed: %v", err)
	}
}

func ExampleClient_Store() {
	var c *imapclient.Client

	seqSet := imap.SeqSetNum(1)
	storeFlags := imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Flags:  []imap.Flag{imap.FlagFlagged},
		Silent: true,
	}
	if err := c.Store(seqSet, &storeFlags, nil).Close(); err != nil {
		log.Fatalf("STORE command failed: %v", err)
	}
}

func ExampleClient_Fetch() {
	var c *imapclient.Client

	seqSet := imap.SeqSetNum(1)
	fetchItems := []imap.FetchItem{
		imap.FetchItemFlags,
		imap.FetchItemEnvelope,
		&imap.FetchItemBodySection{Specifier: imap.PartSpecifierHeader},
	}
	messages, err := c.Fetch(seqSet, fetchItems, nil).Collect()
	if err != nil {
		log.Fatalf("FETCH command failed: %v", err)
	}

	msg := messages[0]
	var header []byte
	for _, buf := range msg.BodySection {
		header = buf
		break
	}

	log.Printf("Flags: %v", msg.Flags)
	log.Printf("Subject: %v", msg.Envelope.Subject)
	log.Printf("Header:\n%v", string(header))
}

func ExampleClient_Fetch_stream() {
	var c *imapclient.Client

	seqSet := imap.SeqSetNum(1)
	fetchItems := []imap.FetchItem{
		imap.FetchItemUID,
		&imap.FetchItemBodySection{},
	}
	fetchCmd := c.Fetch(seqSet, fetchItems, nil)
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		for {
			item := msg.Next()
			if item == nil {
				break
			}

			switch item := item.(type) {
			case imapclient.FetchItemDataUID:
				log.Printf("UID: %v", item.UID)
			case imapclient.FetchItemDataBodySection:
				b, err := io.ReadAll(item.Literal)
				if err != nil {
					log.Fatalf("failed to read body section: %v", err)
				}
				log.Printf("Body:\n%v", string(b))
			}
		}
	}
	if err := fetchCmd.Close(); err != nil {
		log.Fatalf("FETCH command failed: %v", err)
	}
}

func ExampleClient_Search() {
	var c *imapclient.Client

	data, err := c.UIDSearch(&imap.SearchCriteria{
		Body: []string{"Hello world"},
	}, nil).Wait()
	if err != nil {
		log.Fatalf("UID SEARCH command failed: %v", err)
	}
	log.Fatalf("UIDs matching the search criteria: %v", data.AllNums())
}
