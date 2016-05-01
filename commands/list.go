package commands

import (
	imap "github.com/emersion/imap/common"
)

// A LIST command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.8
type List struct {
	Reference string
	Mailbox string
}

func (c *List) Command() *imap.Command {
	return &imap.Command{
		Name: imap.List,
		Arguments: []interface{}{c.Reference, c.Mailbox},
	}
}
