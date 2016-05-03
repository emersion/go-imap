package commands

import (
	"time"

	imap "github.com/emersion/imap/common"
)

// An APPEND command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.11
type Append struct {
	Mailbox string
	Flags []string
	Date *time.Time
	Message *imap.Literal
}

func (c *Append) Command() *imap.Command {
	args := []interface{}{c.Mailbox}

	if c.Flags != nil {
		flags := make([]interface{}, len(c.Flags))
		for i, flag := range c.Flags {
			flags[i] = flag
		}
		args = append(args, flags)
	}

	if c.Date != nil {
		args = append(args, c.Date)
	}

	args = append(args, c.Message)

	return &imap.Command{
		Name: imap.Append,
		Arguments: args,
	}
}
