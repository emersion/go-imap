package commands

import (
	imap "github.com/emersion/imap/common"
)

// An AUTHENTICATE command.
// See https://tools.ietf.org/html/rfc3501#section-6.2.2
type Authenticate struct {
	Mechanism string
}

func (c *Authenticate) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Authenticate,
		Arguments: []interface{}{c.Mechanism},
	}
}

func (c *Authenticate) Parse(fields []interface{}) error {
	c.Mechanism = fields[0].(string)
	return nil
}
