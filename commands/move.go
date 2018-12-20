package commands

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// Move is a MOVE command, as defined in RFC 6851 section 3.1.
type Move struct {
	SeqSet  *imap.SeqSet
	Mailbox string
}

func (cmd *Move) Command() *imap.Command {
	mailbox, _ := utf7.Encoding.NewEncoder().String(cmd.Mailbox)

	return &imap.Command{
		Name:      "MOVE",
		Arguments: []interface{}{cmd.SeqSet, mailbox},
	}
}

func (cmd *Move) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	if seqSet, ok := fields[0].(string); !ok {
		return errors.New("Invalid sequence set")
	} else if seqSet, err := imap.ParseSeqSet(seqSet); err != nil {
		return err
	} else {
		cmd.SeqSet = seqSet
	}

	if mailbox, err := imap.ParseString(fields[1]); err != nil {
		return err
	} else if mailbox, err := utf7.Encoding.NewDecoder().String(mailbox); err != nil {
		return err
	} else {
		cmd.Mailbox = imap.CanonicalMailboxName(mailbox)
	}

	return nil
}
