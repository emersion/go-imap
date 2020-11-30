package commands

import (
	"errors"

	"github.com/emersion/go-imap"
)

// Rename is a RENAME command, as defined in RFC 3501 section 6.3.5.
type Rename struct {
	Existing string
	New      string
}

func (cmd *Rename) Command() *imap.Command {
	existingName := cmd.Existing
	newName := cmd.New

	return &imap.Command{
		Name:      "RENAME",
		Arguments: []interface{}{imap.FormatMailboxName(existingName), imap.FormatMailboxName(newName)},
	}
}

func (cmd *Rename) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	if existingName, err := imap.ParseString(fields[0]); err != nil {
		return err
	} else {
		cmd.Existing = imap.CanonicalMailboxName(existingName)
	}

	if newName, err := imap.ParseString(fields[1]); err != nil {
		return err
	} else {
		cmd.New = imap.CanonicalMailboxName(newName)
	}

	return nil
}
