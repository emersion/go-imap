package commands

import (
	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/utf7"
)

// A CREATE command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.3
type Create struct {
	Mailbox string
}

func (cmd *Create) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name: imap.Create,
		Arguments: []interface{}{mailbox},
	}
}
