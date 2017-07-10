package commands

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// Rename is a RENAME command, as defined in RFC 3501 section 6.3.5.
type Rename struct {
	Existing string
	New      string
}

func (cmd *Rename) Command() *imap.Command {
	existingName, _ := utf7.Encoder.String(cmd.Existing)
	newName, _ := utf7.Encoder.String(cmd.New)

	return &imap.Command{
		Name:      imap.Rename,
		Arguments: []interface{}{existingName, newName},
	}
}

func (cmd *Rename) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	if existingName, err := imap.ParseString(fields[0]); err != nil {
		return err
	} else if existingName, err := utf7.Decoder.String(existingName); err != nil {
		return err
	} else {
		cmd.Existing = imap.CanonicalMailboxName(existingName)
	}

	if newName, err := imap.ParseString(fields[1]); err != nil {
		return err
	} else if newName, err := utf7.Decoder.String(newName); err != nil {
		return err
	} else {
		cmd.New = imap.CanonicalMailboxName(newName)
	}

	return nil
}
