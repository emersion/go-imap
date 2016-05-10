package commands

import (
	imap "github.com/emersion/imap/common"
)

// A CHECK command.
// See https://tools.ietf.org/html/rfc3501#section-6.4.1
type Check struct {}

func (cmd *Check) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Check,
	}
}

func (cmd *Check) Parse(fields []interface{}) error {
	return nil
}
