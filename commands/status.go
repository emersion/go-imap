package commands

import (
	imap "github.com/emersion/imap/common"
)

// A STATUS command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.10
type Status struct {
	Mailbox string
	Items []string
}

func (c *Status) Command() *imap.Command {
	items := make([]interface{}, len(c.Items))
	for i, f := range c.Items {
		items[i] = f
	}

	return &imap.Command{
		Name: imap.Status,
		Arguments: []interface{}{c.Mailbox, items},
	}
}
