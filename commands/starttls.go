package commands

import (
	imap "github.com/emersion/imap/common"
)

// A STARTTLS command.
// See https://tools.ietf.org/html/rfc3501#section-6.2.1
type StartTLS struct {}

func (c *StartTLS) Command() *imap.Command {
	return &imap.Command{
		Name: imap.StartTLS,
	}
}

func (c *StartTLS) Parse(fields []interface{}) error {
	return nil
}
