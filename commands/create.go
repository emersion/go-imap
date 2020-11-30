package commands

import (
	"errors"

	"github.com/emersion/go-imap"
)

// Create is a CREATE command, as defined in RFC 3501 section 6.3.3.
type Create struct {
	Mailbox string
}

func (cmd *Create) Command() *imap.Command {
	mailbox := cmd.Mailbox

	return &imap.Command{
		Name:      "CREATE",
		Arguments: []interface{}{mailbox},
	}
}

func (cmd *Create) Parse(fields []interface{}) error {
	if len(fields) < 1 {
		return errors.New("No enough arguments")
	}

	if mailbox, err := imap.ParseString(fields[0]); err != nil {
		return err
	} else {
		cmd.Mailbox = imap.CanonicalMailboxName(mailbox)
	}

	return nil
}
