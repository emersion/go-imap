package commands

import (
	imap "github.com/emersion/go-imap/common"
)

// A CAPABILITY command.
// See RFC 3501 section 6.1.1
type Capability struct {}

func (c *Capability) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Capability,
	}
}

func (c *Capability) Parse(fields []interface{}) error {
	return nil
}
