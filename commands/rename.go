package commands

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// A RENAME command.
// See RFC 3501 section 6.3.5
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

func (cmd *Rename) Parse(fields []interface{}) (err error) {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	existingName, ok := fields[0].(string)
	if !ok {
		return errors.New("Existing mailbox name must be a string")
	}
	if cmd.Existing, err = utf7.Decoder.String(existingName); err != nil {
		return err
	}

	newName, ok := fields[1].(string)
	if !ok {
		return errors.New("Existing mailbox name must be a string")
	}
	if cmd.New, err = utf7.Decoder.String(newName); err != nil {
		return err
	}

	return
}
