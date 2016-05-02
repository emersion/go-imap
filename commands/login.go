package commands

import (
	imap "github.com/emersion/imap/common"
)

// A LOGIN command.
// See https://tools.ietf.org/html/rfc3501#section-6.2.2
type Login struct {
	Username string
	Password string
}

func (c *Login) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Login,
		Arguments: []interface{}{c.Username, c.Password},
	}
}
