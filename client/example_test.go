package client_test

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/mail"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

func ExampleClient() {
	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS("mail.example.org:993", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login("username", "password"); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// List mailboxes
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Flags for INBOX:", mbox.Flags)

	// Get the last 4 messages
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 3 {
		// We're using unsigned integers here, only substract if the result is > 0
		from = mbox.Messages - 3
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)
	items := []imap.FetchItem{imap.FetchEnvelope}

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	log.Println("Last 4 messages:")
	for msg := range messages {
		log.Println("* " + msg.Envelope.Subject)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	log.Println("Done!")
}

func ExampleClient_Fetch() {
	// Let's assume c is a client
	var c *client.Client

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}

	// Get the last message
	if mbox.Messages == 0 {
		log.Fatal("No message in mailbox")
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(mbox.Messages, mbox.Messages)

	// Get the whole message body
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	log.Println("Last message:")
	msg := <-messages
	r := msg.GetBody(section)
	if r == nil {
		log.Fatal("Server didn't returned message body")
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	m, err := mail.ReadMessage(r)
	if err != nil {
		log.Fatal(err)
	}

	header := m.Header
	log.Println("Date:", header.Get("Date"))
	log.Println("From:", header.Get("From"))
	log.Println("To:", header.Get("To"))
	log.Println("Subject:", header.Get("Subject"))

	body, err := ioutil.ReadAll(m.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(body)
}

func ExampleClient_Append() {
	// Let's assume c is a client
	var c *client.Client

	// Write the message to a buffer
	var b bytes.Buffer
	b.WriteString("From: <root@nsa.gov>\r\n")
	b.WriteString("To: <root@gchq.gov.uk>\r\n")
	b.WriteString("Subject: Hey there\r\n")
	b.WriteString("\r\n")
	b.WriteString("Hey <3")

	// Append it to INBOX, with two flags
	flags := []string{imap.FlaggedFlag, "foobar"}
	if err := c.Append("INBOX", flags, time.Now(), &b); err != nil {
		log.Fatal(err)
	}
}

func ExampleClient_Expunge() {
	// Let's assume c is a client
	var c *client.Client

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}

	// We will delete the last message
	if mbox.Messages == 0 {
		log.Fatal("No message in mailbox")
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(mbox.Messages)

	// First mark the message as deleted
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := c.Store(seqset, item, flags, nil); err != nil {
		log.Fatal(err)
	}

	// Then delete it
	if err := c.Expunge(nil); err != nil {
		log.Fatal(err)
	}

	log.Println("Last message has been deleted")
}

func ExampleClient_StartTLS() {
	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.Dial("mail.example.org:143")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Start a TLS session
	tlsConfig := &tls.Config{ServerName: "mail.example.org"}
	if err := c.StartTLS(tlsConfig); err != nil {
		log.Fatal(err)
	}
	log.Println("TLS started")

	// Now we can login
	if err := c.Login("username", "password"); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")
}

func ExampleClient_Store() {
	// Let's assume c is a client
	var c *client.Client

	// Select INBOX
	_, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}

	// Mark message 42 as seen
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(42)
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.SeenFlag}
	err = c.Store(seqSet, item, flags, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Message has been marked as seen")
}

func ExampleClient_Search() {
	// Let's assume c is a client
	var c *client.Client

	// Select INBOX
	_, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}

	// Set search criteria
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("IDs found:", ids)

	if len(ids) > 0 {
		seqset := new(imap.SeqSet)
		seqset.AddNum(ids...)

		messages := make(chan *imap.Message, 10)
		done := make(chan error, 1)
		go func() {
			done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
		}()

		log.Println("Unseen messages:")
		for msg := range messages {
			log.Println("* " + msg.Envelope.Subject)
		}

		if err := <-done; err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Done!")
}

func ExampleClient_Idle() {
	// Let's assume c is a client
	var c *client.Client

	// Select a mailbox
	if _, err := c.Select("INBOX", false); err != nil {
		log.Fatal(err)
	}

	// Create a channel to receive mailbox updates
	updates := make(chan client.Update)
	c.Updates = updates

	// Start idling
	stopped := false
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- c.Idle(stop, nil)
	}()

	// Listen for updates
	for {
		select {
		case update := <-updates:
			log.Println("New update:", update)
			if !stopped {
				close(stop)
				stopped = true
			}
		case err := <-done:
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Not idling anymore")
			return
		}
	}
}
