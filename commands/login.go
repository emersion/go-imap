package commands

import (
	imap "github.com/emersion/imap/common"
)

// An LOGIN command.
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

func (c *Login) Parse(fields []interface{}) error {
	c.Username = fields[0].(string)
	c.Password = fields[1].(string)
	return nil
}
