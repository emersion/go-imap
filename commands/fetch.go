package commands

import (
	imap "github.com/emersion/imap/common"
)

// A FETCH command.
// See https://tools.ietf.org/html/rfc3501#section-6.4.5
type Fetch struct {
	SeqSet *imap.SeqSet
	Items []string
}

func (c *Fetch) Command() *imap.Command {
	items := make([]interface{}, len(c.Items))
	for i, f := range c.Items {
		items[i] = f
	}

	return &imap.Command{
		Name: imap.Fetch,
		Arguments: []interface{}{c.SeqSet, items},
	}
}
