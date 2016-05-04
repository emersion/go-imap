package commands

import (
	imap "github.com/emersion/imap/common"
)

// A UID command.
// See https://tools.ietf.org/html/rfc3501#section-6.4.8
type Uid struct {
	Cmd imap.Commander
}

func (c *Uid) Command() *imap.Command {
	cmd := c.Cmd.Command()

	args := []interface{}{cmd.Name}
	args = append(args, cmd.Arguments...)

	return &imap.Command{
		Name: imap.Uid,
		Arguments: args,
	}
}
