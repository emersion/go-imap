# imap

[![GoDoc](https://godoc.org/github.com/emersion/imap?status.svg)](https://godoc.org/github.com/emersion/imap)
[![Build Status](https://travis-ci.org/emersion/imap.svg?branch=master)](https://travis-ci.org/emersion/imap)

An [IMAP4rev1](https://tools.ietf.org/html/rfc3501) library written in Go.

```bash
go get gopkg.in/emersion/imap.v0
```

## Why?

Other IMAP implementations in Go:
* Require to make many type assertions
* Are not idiomatic or are [ugly](https://github.com/jordwest/imap-server/blob/master/conn/commands.go#L53)
* Are not pleasant to use
* Implement a server _xor_ a client, not both

## Implemented commands

This package will implement all commands specified in the RFC. Additional
commands will be available in other packages.

Command       | Client | Client tests | Server | Server tests
------------- | ------ | ------------ | ------ | ------------
CAPABILITY    | ✓      | ✓            | ✓      | ✗
NOOP          | ✓      | ✓            | ✓      | ✗
LOGOUT        | ✓      | ✓            | ✓      | ✗
AUTHENTICATE  | ✓      | ✓            | ✓      | ✗
LOGIN         | ✓      | ✓            | ✓      | ✗
STARTTLS      | ✓      | ✗            | ✗      | ✗
SELECT        | ✓      | ✓            | ✓      | ✗
EXAMINE       | ✓      | ✓            | ✓      | ✗
CREATE        | ✓      | ✓            | ✗      | ✗
DELETE        | ✓      | ✓            | ✗      | ✗
RENAME        | ✓      | ✓            | ✗      | ✗
SUBSCRIBE     | ✓      | ✓            | ✗      | ✗
UNSUBSCRIBE   | ✓      | ✓            | ✗      | ✗
LIST          | ✓      | ✓            | ✓      | ✗
LSUB          | ✓      | ✓            | ✗      | ✗
STATUS        | ✓      | ✓            | ✓      | ✗
APPEND        | ✓      | ✓            | ✗      | ✗
CHECK         | ✓      | ✓            | ✗      | ✗
CLOSE         | ✓      | ✓            | ✓      | ✗
EXPUNGE       | ✓      | ✓            | ✗      | ✗
SEARCH        | ✓      | ✓            | ✓      | ✗
FETCH         | ✓      | ✓            | ✓      | ✗
STORE         | ✓      | ✓            | ✗      | ✗
COPY          | ✓      | ✓            | ✗      | ✗
UID           | ✓      | ✗            | ✗      | ✗

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

	// Don't forget to logout
	defer c.Logout()

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
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Flags for INBOX:", mbox.Flags)

	// Get the last 4 messages
	seqset, _ := imap.NewSeqSet("")
	seqset.AddRange(mbox.Messages - 3, mbox.Messages)

	messages := make(chan *imap.Message, 4)
	err = c.Fetch(seqset, []string{"ENVELOPE"}, messages)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range messages {
		log.Println(msg.Envelope.Subject)
	}

	log.Println("Done!")
}
```

## License

MIT
