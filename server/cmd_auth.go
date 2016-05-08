package server

import (
	"errors"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
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

type List struct {
	commands.List
}

func (cmd *List) Handle(conn *Conn) error {
	if conn.User == nil {
		return errors.New("Not authenticated")
	}

	done := make(chan error)
	defer close(done)

	ch := make(chan *common.MailboxInfo)

	res := responses.List{Mailboxes: ch}

	go (func () {
		done <- res.WriteTo(conn.Writer)
	})()

	mailboxes, err := conn.User.ListMailboxes()
	if err != nil {
		close(ch)
		return err
	}

	for _, mbox := range mailboxes {
		// TODO: filter mailboxes with cmd.Reference and cmd.Mailbox

		info, err := mbox.Info()
		if err != nil {
			close(ch)
			return err
		}

		ch <- info
	}

	close(ch)

	return <-done
}
