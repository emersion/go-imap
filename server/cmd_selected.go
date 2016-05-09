package server

import (
	"errors"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
)

// A command handler that supports UIDs.
type UidHandler interface {
	Handler

	// Handle this command using UIDs for a given connection.
	UidHandle(conn *Conn) error
}

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

type Search struct {
	commands.Search
}

func (cmd *Search) handle(uid bool, conn *Conn) error {
	if conn.Mailbox == nil {
		return errors.New("No mailbox selected")
	}

	ids, err := conn.Mailbox.SearchMessages(uid, cmd.Criteria)
	if err != nil {
		return err
	}

	res := responses.Search{Ids: ids}
	return res.WriteTo(conn.Writer)
}

func (cmd *Search) Handle(conn *Conn) error {
	return cmd.handle(false, conn)
}

func (cmd *Search) UidHandle(conn *Conn) error {
	return cmd.handle(true, conn)
}

type Fetch struct {
	commands.Fetch
}

func (cmd *Fetch) handle(uid bool, conn *Conn) error {
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

	messages, err := conn.Mailbox.ListMessages(uid, cmd.SeqSet, cmd.Items)
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

func (cmd *Fetch) Handle(conn *Conn) error {
	return cmd.handle(false, conn)
}

func (cmd *Fetch) UidHandle(conn *Conn) error {
	// Append UID to the list of requested items if it isn't already present
	hasUid := false
	for _, item := range cmd.Items {
		if item == "UID" {
			hasUid = true
			break
		}
	}
	if !hasUid {
		cmd.Items = append(cmd.Items, "UID")
	}

	return cmd.handle(true, conn)
}

type Uid struct {
	commands.Uid
}

func (cmd *Uid) Handle(conn *Conn) error {
	hdlr, err := conn.Server.getCommandHandler(cmd.Cmd.Command())
	if err != nil {
		return err
	}

	uidHdlr, ok := hdlr.(UidHandler)
	if !ok {
		return errors.New("Command unsupported with UID")
	}

	return uidHdlr.UidHandle(conn)
}
