package commands

import (
	imap "github.com/emersion/imap/common"
)

// A CREATE command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.3
type Create struct {
	Mailbox string
}

func (c *Create) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Create,
		Arguments: []interface{}{c.Mailbox},
	}
}
