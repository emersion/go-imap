package commands

import (
	imap "github.com/emersion/imap/common"
)

// A CLOSE command.
// See https://tools.ietf.org/html/rfc3501#section-6.4.2
type Close struct {}

func (c *Close) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Close,
	}
}
