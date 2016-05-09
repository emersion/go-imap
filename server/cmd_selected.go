package server

import (
	"errors"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
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

type Fetch struct {
	commands.Fetch
}

func (cmd *Fetch) Handle(conn *Conn) error {
	if conn.Mailbox == nil {
		return errors.New("No mailbox selected")
	}

	done := make(chan error)
	defer close(done)

	ch := make(chan *common.Message)
	res := responses.Fetch{Messages: ch}

	go (func () {
		done <- res.WriteTo(conn.Writer)
	})()

	messages, err := conn.Mailbox.ListMessages(false, cmd.SeqSet, cmd.Items)
	if err != nil {
		close(ch)
		return err
	}

	for _, msg := range messages {
		ch <- msg
	}

	close(ch)

	return <-done
}
