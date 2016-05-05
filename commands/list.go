package commands

import (
	imap "github.com/emersion/imap/common"
)

// A LIST command.
// If Subscribed is set to true, LSUB will be used instead.
// See https://tools.ietf.org/html/rfc3501#section-6.3.8
type List struct {
	Reference string
	Mailbox string

	Subscribed bool
}

func (c *List) Command() *imap.Command {
	name := imap.List
	if c.Subscribed {
		name = imap.Lsub
	}

	return &imap.Command{
		Name: name,
		Arguments: []interface{}{c.Reference, c.Mailbox},
	}
}
