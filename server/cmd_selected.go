package server

import (
	"errors"

	//"github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	//"github.com/emersion/imap/responses"
)

type Close struct {
	commands.Close
}

func (cmd *Close) Handle(conn *Conn) error {
	if conn.Mailbox == nil {
		return errors.New("No mailbox selected")
	}

	conn.Mailbox = nil
	conn.MailboxReadOnly = false
	return nil
}
