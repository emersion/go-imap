# imap

[![GoDoc](https://godoc.org/github.com/emersion/imap?status.svg)](https://godoc.org/github.com/emersion/imap)

An IMAP library written in Go.

```bash
go get gopkg.in/emersion/imap.v0
```

## Why?

Other IMAP implementations in Go:
* Require to make many type assertions
* Are not idiomatic

## Usage

```go
import (
	"log"

	"github.com/emersion/imap/client"
	imap "github.com/emersion/imap/common"
)

func main() {
	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS("mail.example.org:993", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Login
	if err := c.Login("username", "password"); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// List mailboxes
	mailboxes := make(chan *imap.MailboxInfo)
	go (func () {
		err = c.List("", "%", mailboxes)
	})()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println(m.Name)
	}

	if err != nil {
		log.Fatal(err)
	}

	// Select INBOX
	mbox, err := c.Select("INBOX")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Flags for INBOX:", mbox.Flags)

	// Get the last 4 messages
	seqset, _ := imap.NewSeqSet("")
	seqset.AddRange(mbox.Total - 3, mbox.Total)

	messages := make(chan *imap.Message, 4)
	err = c.Fetch(seqset, []string{"ENVELOPE"}, messages)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range messages {
		log.Println(msg.Envelope().Subject)
	}

	log.Println("Done!")
}
```

## License

MIT
