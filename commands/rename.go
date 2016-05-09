package commands

import (
	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/utf7"
)

// A RENAME command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.5
type Rename struct {
	Existing string
	New string
}

func (cmd *Rename) Command() *imap.Command {
	existingName, _ := utf7.Encoder.String(cmd.Existing)
	newName, _ := utf7.Encoder.String(cmd.New)

	return &imap.Command{
		Name: imap.Rename,
		Arguments: []interface{}{existingName, newName},
	}
}
