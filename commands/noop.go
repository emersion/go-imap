package commands

import (
	imap "github.com/emersion/go-imap/common"
)

// A NOOP command.
// See RFC 3501 section 6.1.2
type Noop struct {}

func (c *Noop) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Noop,
	}
}

func (c *Noop) Parse(fields []interface{}) error {
	return nil
}
