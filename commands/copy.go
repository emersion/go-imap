package commands

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// Copy is a COPY command, as defined in RFC 3501 section 6.4.7.
type Copy struct {
	SeqSet  *imap.SeqSet
	Mailbox string
}

func (cmd *Copy) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name:      imap.Copy,
		Arguments: []interface{}{cmd.SeqSet, mailbox},
	}
}

func (cmd *Copy) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	if seqSet, ok := fields[0].(string); !ok {
		return errors.New("Invalid sequence set")
	} else if seqSet, err := imap.NewSeqSet(seqSet); err != nil {
		return err
	} else {
		cmd.SeqSet = seqSet
	}

	if mailbox, ok := fields[1].(string); !ok {
		return errors.New("Mailbox name must be a string")
	} else if mailbox, err := utf7.Decoder.String(mailbox); err != nil {
		return err
	} else {
		cmd.Mailbox = imap.CanonicalMailboxName(mailbox)
	}

	return nil
}
