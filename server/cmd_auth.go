package server

import (
	"errors"

	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
)

// Common errors in Authenticated state.
var (
	ErrNotAuthenticated = errors.New("Not authenticated")
)

type Select struct {
	commands.Select
}

func (cmd *Select) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := conn.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		return err
	}

	items := []string{
		common.MailboxFlags, common.MailboxPermanentFlags,
		common.MailboxMessages, common.MailboxRecent, common.MailboxUnseen,
		common.MailboxUidNext, common.MailboxUidValidity,
	}

	status, err := mbox.Status(items)
	if err != nil {
		return err
	}

	conn.Mailbox = mbox
	conn.MailboxReadOnly = cmd.ReadOnly || status.ReadOnly

	res := &responses.Select{Mailbox: status}
	return conn.WriteResp(res)
}

type Create struct {
	commands.Create
}

func (cmd *Create) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	return conn.User.CreateMailbox(cmd.Mailbox)
}

type Delete struct {
	commands.Delete
}

func (cmd *Delete) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	return conn.User.DeleteMailbox(cmd.Mailbox)
}

type Rename struct {
	commands.Rename
}

func (cmd *Rename) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	return conn.User.RenameMailbox(cmd.Existing, cmd.New)
}

type Subscribe struct {
	commands.Subscribe
}

func (cmd *Subscribe) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := conn.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		return err
	}

	return mbox.Subscribe()
}

type Unsubscribe struct {
	commands.Unsubscribe
}

func (cmd *Unsubscribe) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := conn.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		return err
	}

	return mbox.Unsubscribe()
}

type List struct {
	commands.List
}

func (cmd *List) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	done := make(chan error)
	defer close(done)

	ch := make(chan *common.MailboxInfo)
	res := &responses.List{Mailboxes: ch, Subscribed: cmd.Subscribed}

	go (func () {
		done <- conn.WriteResp(res)
	})()

	mailboxes, err := conn.User.ListMailboxes(cmd.Subscribed)
	if err != nil {
		close(ch)
		return err
	}

	for _, mbox := range mailboxes {
		info, err := mbox.Info()
		if err != nil {
			close(ch)
			return err
		}

		// TODO: filter mailboxes with cmd.Reference and cmd.Mailbox
		if cmd.Reference != "" || (cmd.Mailbox != "*" && cmd.Mailbox != "%" && cmd.Mailbox != info.Name) {
			continue
		}

		ch <- info
	}

	close(ch)

	return <-done
}

type Status struct {
	commands.Status
}

func (cmd *Status) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := conn.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		return err
	}

	status, err := mbox.Status(cmd.Items)
	if err != nil {
		return err
	}

	res := &responses.Status{Mailbox: status}
	return conn.WriteResp(res)
}

type Append struct {
	commands.Append
}

func (cmd *Append) Handle(conn *Conn) error {
	if conn.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := conn.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		// TODO: add [TRYCREATE] to the NO response
		return err
	}

	if err := mbox.CreateMessage(cmd.Flags, cmd.Date, cmd.Message.Bytes()); err != nil {
		return err
	}

	// If APPEND targets the currently selected mailbox, send an untagged EXISTS
	// Do this only if the backend doesn't send updates itself
	if conn.Server.Updates == nil && conn.Mailbox != nil && conn.Mailbox.Name() == mbox.Name() {
		status, err := mbox.Status([]string{common.MailboxMessages})
		if err != nil {
			return err
		}

		res := &responses.Select{Mailbox: status}
		if err := conn.WriteResp(res); err != nil {
			return err
		}
	}

	return nil
}
