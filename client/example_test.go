package client_test

import (
	"io/ioutil"
	"log"
	"net/mail"

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
	go func() {
		// c.List will send mailboxes to the channel and close it when done
		if err := c.List("", "*", mailboxes); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
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

	messages := make(chan *imap.Message, 10)
	go func() {
		if err := c.Fetch(seqset, []string{imap.EnvelopeMsgAttr}, messages); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("Last 4 messages:")
	for msg := range messages {
		log.Println("* " + msg.Envelope.Subject)
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
	attrs := []string{"BODY[]"}

	messages := make(chan *imap.Message, 1)
	go func() {
		if err := c.Fetch(seqset, attrs, messages); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("Last message:")
	msg := <-messages
	r := msg.GetBody("BODY[]")
	if r == nil {
		log.Fatal("Server didn't returned message body")
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
	seqset.AddRange(mbox.Messages, mbox.Messages)

	// First mark the message as deleted
	operation := "+FLAGS.SILENT"
	flags := []interface{}{imap.DeletedFlag}
	if err := c.Store(seqset, operation, flags, nil); err != nil {
		log.Fatal(err)
	}

	// Then delete it
	if err := c.Expunge(nil); err != nil {
		log.Fatal(err)
	}

	log.Println("Last message has been deleted")
}
