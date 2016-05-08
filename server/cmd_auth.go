package server

import (
	"errors"

	"github.com/emersion/imap/commands"
)

type Select struct {
	commands.Select
}

func (cmd *Select) Handle(conn *Conn) error {
	if conn.User == nil {
		return errors.New("Not authenticated")
	}

	mailbox, err := conn.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		return err
	}

	conn.Mailbox = mailbox
	conn.MailboxReadOnly = cmd.ReadOnly
	return nil
}
