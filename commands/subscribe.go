package commands

import (
	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/utf7"
)

// A SUBSCRIBE command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.6
type Subscribe struct {
	Mailbox string
}

func (cmd *Subscribe) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name: imap.Subscribe,
		Arguments: []interface{}{mailbox},
	}
}

// An UNSUBSCRIBE command.
// See https://tools.ietf.org/html/rfc3501#section-6.3.7
type Unsubscribe struct {
	Mailbox string
}

func (cmd *Unsubscribe) Command() *imap.Command {
	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)

	return &imap.Command{
		Name: imap.Unsubscribe,
		Arguments: []interface{}{mailbox},
	}
}
