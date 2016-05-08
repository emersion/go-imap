package commands

import (
	"errors"

	imap "github.com/emersion/imap/common"
)

// A LIST command.
// If Subscribed is set to true, LSUB will be used instead.
// See https://tools.ietf.org/html/rfc3501#section-6.3.8
type List struct {
	Reference string
	Mailbox string

	Subscribed bool
}

func (cmd *List) Command() *imap.Command {
	name := imap.List
	if cmd.Subscribed {
		name = imap.Lsub
	}

	return &imap.Command{
		Name: name,
		Arguments: []interface{}{cmd.Reference, cmd.Mailbox},
	}
}

func (cmd *List) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	var ok bool
	if cmd.Reference, ok = fields[0].(string); !ok {
		return errors.New("Reference must be a string")
	}
	if cmd.Mailbox, ok = fields[1].(string); !ok {
		return errors.New("Mailbox must be a string")
	}

	return nil
}
