package commands

import (
	imap "github.com/emersion/imap/common"
)

// A STARTTLS command.
// See https://tools.ietf.org/html/rfc3501#section-6.2.1
type Starttls struct {}

func (c *Starttls) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Starttls,
	}
}

func (c *Starttls) Parse(fields []interface{}) error {
	return nil
}
