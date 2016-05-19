package commands

import (
	imap "github.com/emersion/go-imap/common"
)

// A CHECK command.
// See RFC 3501 section 6.4.1
type Check struct {}

func (cmd *Check) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Check,
	}
}

func (cmd *Check) Parse(fields []interface{}) error {
	return nil
}
