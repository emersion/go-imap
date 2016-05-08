package commands

import (
	"errors"

	imap "github.com/emersion/imap/common"
)

// A SELECT command.
// If ReadOnly is set to true, the EXAMINE command will be used instead.
// See https://tools.ietf.org/html/rfc3501#section-6.3.1
type Select struct {
	Mailbox string
	ReadOnly bool
}

func (cmd *Select) Command() *imap.Command {
	name := imap.Select
	if cmd.ReadOnly {
		name = imap.Examine
	}

	return &imap.Command{
		Name: name,
		Arguments: []interface{}{cmd.Mailbox},
	}
}

func (cmd *Select) Parse(fields []interface{}) error {
	if len(fields) < 1 {
		return errors.New("No enough arguments")
	}

	var ok bool
	if cmd.Mailbox, ok = fields[0].(string); !ok {
		return errors.New("Mailbox name is not a string")
	}

	return nil
}
