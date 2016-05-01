package commands

import (
	imap "github.com/emersion/imap/common"
)

// A SELECT command.
// If ReadOnly is set to true, the EXAMINE command will be used instead.
// See https://tools.ietf.org/html/rfc3501#section-6.3.1
type Select struct {
	Mailbox string
	ReadOnly bool
}

func (c *Select) Command() *imap.Command {
	name := imap.Select
	if c.ReadOnly {
		name = imap.Examine
	}

	return &imap.Command{
		Name: name,
		Arguments: []interface{}{c.Mailbox},
	}
}
