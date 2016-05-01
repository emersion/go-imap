package commands

import (
	imap "github.com/emersion/imap/common"
)

// A SELECT command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.1
type Select struct {
	Mailbox string
}

func (c *Select) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Select,
		Arguments: []interface{}{c.Mailbox},
	}
}
