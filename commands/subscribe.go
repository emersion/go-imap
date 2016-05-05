package commands

import (
	imap "github.com/emersion/imap/common"
)

// A SUBSCRIBE command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.6
type Subscribe struct {
	Mailbox string
}

func (c *Subscribe) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Subscribe,
		Arguments: []interface{}{c.Mailbox},
	}
}

// An UNSUBSCRIBE command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.7
type Unsubscribe struct {
	Mailbox string
}

func (c *Unsubscribe) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Unsubscribe,
		Arguments: []interface{}{c.Mailbox},
	}
}
