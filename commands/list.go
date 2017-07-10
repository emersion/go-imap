package commands

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// List is a LIST command, as defined in RFC 3501 section 6.3.8. If Subscribed
// is set to true, LSUB will be used instead.
type List struct {
	Reference string
	Mailbox   string

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
		Name:      name,
		Arguments: []interface{}{ref, mailbox},
	}
}

func (cmd *List) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	if mailbox, err := imap.ParseString(fields[0]); err != nil {
		return err
	} else if mailbox, err := utf7.Decoder.String(mailbox); err != nil {
		return err
	} else {
		// TODO: canonical mailbox path
		cmd.Reference = imap.CanonicalMailboxName(mailbox)
	}

	if mailbox, err := imap.ParseString(fields[1]); err != nil {
		return err
	} else if mailbox, err := utf7.Decoder.String(mailbox); err != nil {
		return err
	} else {
		cmd.Mailbox = imap.CanonicalMailboxName(mailbox)
	}

	return nil
}
