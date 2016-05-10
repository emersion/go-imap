package commands

import (
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/utf7"
)

// A CREATE command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.3
type Create struct {
	Mailbox string
}

func (cmd *Create) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name: imap.Create,
		Arguments: []interface{}{mailbox},
	}
}

func (cmd *Create) Parse(fields []interface{}) (err error) {
	if len(fields) < 1 {
		return errors.New("No enough arguments")
	}

	mailbox, ok := fields[0].(string)
	if !ok {
		return errors.New("Mailbox must be a string")
	}
	if cmd.Mailbox, err = utf7.Decoder.String(mailbox); err != nil {
		return err
	}

	return
}
