package commands

import (
	"github.com/emersion/go-imap"
)

// An EXPUNGE command.
// See RFC 3501 section 6.4.3
type Expunge struct {}

func (cmd *Expunge) Command() *imap.Command {
	return &imap.Command{Name: imap.Expunge}
}

func (cmd *Expunge) Parse(fields []interface{}) error {
	return nil
}
