package commands

import (
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/utf7"
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

	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name: name,
		Arguments: []interface{}{mailbox},
	}
}

func (cmd *Select) Parse(fields []interface{}) error {
	if len(fields) < 1 {
		return errors.New("No enough arguments")
	}

	mailbox, ok := fields[0].(string)
	if !ok {
		return errors.New("Mailbox name is not a string")
	}

	var err error
	if cmd.Mailbox, err = utf7.Decoder.String(mailbox); err != nil {
		return err
	}

	return nil
}
