package commands

import (
	imap "github.com/emersion/imap/common"
)

// A STORE command.
// See https://tools.ietf.org/html/rfc3501#section-6.4.6
type Store struct {
	SeqSet *imap.SeqSet
	Item string
	Value interface{}
}

func (c *Store) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Store,
		Arguments: []interface{}{c.SeqSet, c.Item, c.Value},
	}
}
