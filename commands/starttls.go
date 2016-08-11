package commands

import (
	"github.com/emersion/go-imap"
)

// A STARTTLS command.
// See RFC 3501 section 6.2.1
type StartTLS struct{}

func (cmd *StartTLS) Command() *imap.Command {
	return &imap.Command{
		Name: imap.StartTLS,
	}
}

func (cmd *StartTLS) Parse(fields []interface{}) error {
	return nil
}
