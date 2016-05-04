package commands

import (
	imap "github.com/emersion/imap/common"
)

// A RENAME command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.5
type Rename struct {
	Existing string
	New string
}

func (c *Rename) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Rename,
		Arguments: []interface{}{c.Existing, c.New},
	}
}
