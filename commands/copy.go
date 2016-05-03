package commands

import (
	imap "github.com/emersion/imap/common"
)

// A COPY command.
// See https://tools.ietf.org/html/rfc3501#section-6.4.7
type Copy struct {
	SeqSet *imap.SeqSet
	Mailbox string
}

func (c *Copy) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Copy,
		Arguments: []interface{}{c.SeqSet, c.Mailbox},
	}
}
