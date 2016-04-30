package commands

import (
	imap "github.com/emersion/imap/common"
)

// A LOGOUT command.
// See https://tools.ietf.org/html/rfc3501#section-6.1.3
type Logout struct {}

func (c *Logout) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Logout,
	}
}
