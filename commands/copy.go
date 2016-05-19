package commands

import (
	"errors"

	imap "github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/utf7"
)

// A COPY command.
// See RFC 3501 section 6.4.7
type Copy struct {
	SeqSet *imap.SeqSet
	Mailbox string
}

func (cmd *Copy) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name: imap.Copy,
		Arguments: []interface{}{cmd.SeqSet, mailbox},
	}
}

func (cmd *Copy) Parse(fields []interface{}) (err error) {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	seqset, ok := fields[0].(string)
	if !ok {
		return errors.New("Invaliud sequence set")
	}
	if cmd.SeqSet, err = imap.NewSeqSet(seqset); err != nil {
		return err
	}

	mailbox, ok := fields[1].(string)
	if !ok {
		return errors.New("Mailbox name must be a string")
	}
	if cmd.Mailbox, err = utf7.Decoder.String(mailbox); err != nil {
		return err
	}

	return
}
