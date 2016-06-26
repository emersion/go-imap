# go-imap

[![GoDoc](https://godoc.org/github.com/emersion/go-imap?status.svg)](https://godoc.org/github.com/emersion/go-imap)
[![Build Status](https://travis-ci.org/emersion/go-imap.svg?branch=master)](https://travis-ci.org/emersion/go-imap)
[![codecov](https://codecov.io/gh/emersion/go-imap/branch/master/graph/badge.svg)](https://codecov.io/gh/emersion/go-imap)
![stability-unstable](https://img.shields.io/badge/stability-unstable-yellow.svg)

An [IMAP4rev1](https://tools.ietf.org/html/rfc3501) library written in Go. It
can be used to build a client and/or a server and supports UTF-7.

```bash
go get github.com/emersion/go-imap
```

## Why?

Other IMAP implementations in Go:
* Require to make [many type assertions or conversions](https://github.com/emersion/neutron/blob/ca635850e2223d6cfe818664ef901fa6e3c1d859/backend/imap/util.go#L110)
* Are not idiomatic or are [ugly](https://github.com/jordwest/imap-server/blob/master/conn/commands.go#L53)
* Are [not pleasant to use](https://github.com/emersion/neutron/blob/ca635850e2223d6cfe818664ef901fa6e3c1d859/backend/imap/messages.go#L228)
* Implement a server _xor_ a client, not both
* Don't implement unilateral updates (i.e. the server can't notify clients for
  new messages)

## Implemented commands

This package implements all commands specified in the RFC.

Command       | Client | Client tests | Server | Server tests
------------- | ------ | ------------ | ------ | ------------
CAPABILITY    | ✓      | ✓            | ✓      | ✓
NOOP          | ✓      | ✓            | ✓      | ✓
LOGOUT        | ✓      | ✓            | ✓      | ✓
AUTHENTICATE  | ✓      | ✓            | ✓      | ✓
LOGIN         | ✓      | ✓            | ✓      | ✓
STARTTLS      | ✓      | ✗            | ✓      | ✗
SELECT        | ✓      | ✓            | ✓      | ✓
EXAMINE       | ✓      | ✓            | ✓      | ✗
CREATE        | ✓      | ✓            | ✓      | ✓
DELETE        | ✓      | ✓            | ✓      | ✓
RENAME        | ✓      | ✓            | ✓      | ✓
SUBSCRIBE     | ✓      | ✓            | ✓      | ✓
UNSUBSCRIBE   | ✓      | ✓            | ✓      | ✓
LIST          | ✓      | ✓            | ✓      | ✓
LSUB          | ✓      | ✓            | ✓      | ✗
STATUS        | ✓      | ✓            | ✓      | ✓
APPEND        | ✓      | ✗            | ✓      | ✗
CHECK         | ✓      | ✓            | ✓      | ✗
CLOSE         | ✓      | ✓            | ✓      | ✗
EXPUNGE       | ✓      | ✓            | ✓      | ✗
SEARCH        | ✓      | ✓            | ✓      | ✗
FETCH         | ✓      | ✓            | ✓      | ✗
STORE         | ✓      | ✓            | ✓      | ✗
COPY          | ✓      | ✓            | ✓      | ✗
UID           | ✓      | ✗            | ✓      | ✗

## IMAP extensions

Commands defined in IMAP extensions are available in other packages.

* [COMPRESS](https://github.com/emersion/go-imap-compress)
* [SPECIAL-USE](https://github.com/emersion/go-imap-specialuse)
* [MOVE](https://github.com/emersion/go-imap-move)

## Usage

### Client

```go
package main

import (
	"log"

	"github.com/emersion/go-imap/client"
	imap "github.com/emersion/go-imap/common"
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
		log.Println("* " + m.Name)
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

### Server

```go
package main

import (
	"log"

	"github.com/emersion/go-imap/server"
	"github.com/emersion/go-imap/backend/memory"
)

func main() {
	// Create a memory backend
	bkd := memory.New()

	// Create a new server
	s, err := server.Listen(":3000", bkd)
	if err != nil {
		log.Fatal(err)
	}

	// Since we will use this server for testing only, we can allow plain text
	// authentication over unencrypted connections
	s.AllowInsecureAuth = true

	log.Println("Server listening at", s.Addr())

	// Do something else to keep the server alive
	select {}
}
```

You can now use `telnet localhost 3000` to manually connect to the server.

## License

MIT
