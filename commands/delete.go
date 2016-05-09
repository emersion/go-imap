package commands

import (
	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/utf7"
)

// A DELETE command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.3
type Delete struct {
	Mailbox string
}

func (cmd *Delete) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name: imap.Delete,
		Arguments: []interface{}{mailbox},
	}
}
