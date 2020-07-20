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
	mailbox, _ := utf7.Encoding.NewEncoder().String(cmd.Mailbox)

	return &imap.Command{
		Name:      "SUBSCRIBE",
		Arguments: []interface{}{imap.FormatMailboxName(mailbox)},
	}
}

func (cmd *Subscribe) Parse(fields []interface{}) error {
	if len(fields) == 0 {
		return errors.New("not enough arguments")
	}

	if mailbox, err := imap.ParseString(fields[0]); err != nil {
		return err
	} else if cmd.Mailbox, err = utf7.Encoding.NewDecoder().String(mailbox); err != nil {
		return err
	}
	return nil
}

// Unsubscribe is an UNSUBSCRIBE command, as defined in RFC 3501 section 6.3.7.
type Unsubscribe struct {
	Mailbox string
}

func (cmd *Unsubscribe) Command() *imap.Command {
	mailbox, _ := utf7.Encoding.NewEncoder().String(cmd.Mailbox)

	return &imap.Command{
		Name:      "UNSUBSCRIBE",
		Arguments: []interface{}{imap.FormatMailboxName(mailbox)},
	}
}

func (cmd *Unsubscribe) Parse(fields []interface{}) error {
	if len(fields) == 0 {
		return errors.New("not enogh arguments")
	}

	if mailbox, err := imap.ParseString(fields[0]); err != nil {
		return err
	} else if cmd.Mailbox, err = utf7.Encoding.NewDecoder().String(mailbox); err != nil {
		return err
	}
	return nil
}
