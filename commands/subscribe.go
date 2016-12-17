package commands

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// Subscribe is a SUBSCRIBE command, as defined in RFC 3501 section 6.3.6.
type Subscribe struct {
	Mailbox string
}

func (cmd *Subscribe) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name:      imap.Subscribe,
		Arguments: []interface{}{mailbox},
	}
}

func (cmd *Subscribe) Parse(fields []interface{}) (err error) {
	if len(fields) < 0 {
		return errors.New("No enogh arguments")
	}

	mailbox, ok := fields[0].(string)
	if !ok {
		return errors.New("Mailbox name must be a string")
	}

	if cmd.Mailbox, err = utf7.Decoder.String(mailbox); err != nil {
		return err
	}

	return
}

// An UNSUBSCRIBE command.
// See RFC 3501 section 6.3.7
type Unsubscribe struct {
	Mailbox string
}

func (cmd *Unsubscribe) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name:      imap.Unsubscribe,
		Arguments: []interface{}{mailbox},
	}
}

func (cmd *Unsubscribe) Parse(fields []interface{}) (err error) {
	if len(fields) < 0 {
		return errors.New("No enogh arguments")
	}

	mailbox, ok := fields[0].(string)
	if !ok {
		return errors.New("Mailbox name must be a string")
	}

	if cmd.Mailbox, err = utf7.Decoder.String(mailbox); err != nil {
		return err
	}

	return
}
