package commands

import (
	"errors"

	"github.com/emersion/go-imap"
)

// Expunge is an EXPUNGE command, as defined in RFC 3501 section 6.4.3.
type Expunge struct {
	// UID seqset specified for UID EXPUNGE (UIDPLUS extension).
	SeqSet *imap.SeqSet
}

func (cmd *Expunge) Command() *imap.Command {
	return &imap.Command{Name: "EXPUNGE"}
}

func (cmd *Expunge) Parse(fields []interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	seqSet, ok := fields[0].(string)
	if !ok {
		return errors.New("String required as an argument")
	}
	var err error
	cmd.SeqSet, err = imap.ParseSeqSet(seqSet)
	return err
}
