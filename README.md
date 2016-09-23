# go-imap

[![GoDoc](https://godoc.org/github.com/emersion/go-imap?status.svg)](https://godoc.org/github.com/emersion/go-imap)
[![Build Status](https://travis-ci.org/emersion/go-imap.svg?branch=master)](https://travis-ci.org/emersion/go-imap)
[![codecov](https://codecov.io/gh/emersion/go-imap/branch/master/graph/badge.svg)](https://codecov.io/gh/emersion/go-imap)
[![stability-unstable](https://img.shields.io/badge/stability-unstable-yellow.svg)](https://github.com/emersion/stability-badges#unstable)

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
* Do not have a good test coverage

## Implemented commands

This package implements all commands specified in the RFC. Each command has its
own tests.

## IMAP extensions

Commands defined in IMAP extensions are available in other packages. See [the
wiki](https://github.com/emersion/go-imap/wiki/Using-extensions#using-client-extensions)
to see how to use them.

* [APPENDLIMIT](https://github.com/emersion/go-imap-appendlimit)
* [COMPRESS](https://github.com/emersion/go-imap-compress)
* [ID](https://github.com/ProtonMail/go-imap-id)
* [IDLE](https://github.com/emersion/go-imap-idle)
* [MOVE](https://github.com/emersion/go-imap-move)
* [QUOTA](https://github.com/emersion/go-imap-quota)
* [SPECIAL-USE](https://github.com/emersion/go-imap-specialuse)
* [UNSELECT](https://github.com/emersion/go-imap-unselect)

## Usage

### Client

```go
package main

import (
	"log"

	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap"
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
	done := make(chan error, 1)
	go func () {
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
	seqset, _ := imap.NewSeqSet("")
	seqset.AddRange(mbox.Messages - 3, mbox.Messages)

	messages := make(chan *imap.Message)
	done = make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []string{imap.EnvelopeMsgAttr}, messages)
	}()

	for msg := range messages {
		log.Println(msg.Envelope.Subject)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
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
	s := server.New(bkd)
	s.Addr = ":3000"
	// Since we will use this server for testing only, we can allow plain text
	// authentication over unencrypted connections
	s.AllowInsecureAuth = true

	log.Println("Starting IMAP server at localhost:3000")
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
```

You can now use `telnet localhost 3000` to manually connect to the server.

## License

MIT
