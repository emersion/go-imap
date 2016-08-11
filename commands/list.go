package commands

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// A LIST command.
// If Subscribed is set to true, LSUB will be used instead.
// See RFC 3501 section 6.3.8
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

	ref, _ := utf7.Encoder.String(cmd.Reference)
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name: name,
		Arguments: []interface{}{ref, mailbox},
	}
}

func (cmd *List) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	ref, ok := fields[0].(string)
	if !ok {
		return errors.New("Reference must be a string")
	}

	mailbox, ok := fields[1].(string)
	if !ok {
		return errors.New("Mailbox must be a string")
	}

	var err error
	if cmd.Reference, err = utf7.Decoder.String(ref); err != nil {
		return err
	}
	if cmd.Mailbox, err = utf7.Decoder.String(mailbox); err != nil {
		return err
	}

	return nil
}
