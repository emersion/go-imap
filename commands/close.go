package commands

import (
	imap "github.com/emersion/imap/common"
)

// A CLOSE command.
// See RFC 3501 section 6.4.2
type Close struct {}

func (cmd *Close) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Close,
	}
}

func (cmd *Close) Parse(fields []interface{}) error {
	return nil
}
