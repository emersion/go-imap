package server

import (
	"errors"
	"strings"

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

func (cmd *Select) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := ctx.User.GetMailbox(cmd.Mailbox)
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

	ctx.Mailbox = mbox
	ctx.MailboxReadOnly = cmd.ReadOnly || status.ReadOnly

	res := &responses.Select{Mailbox: status}
	if err := conn.WriteResp(res); err != nil {
		return err
	}

	code := common.CodeReadWrite
	if ctx.MailboxReadOnly {
		code = common.CodeReadOnly
	}
	return ErrStatusResp(&common.StatusResp{
		Type: common.StatusOk,
		Code: code,
	})
}

type Create struct {
	commands.Create
}

func (cmd *Create) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	return ctx.User.CreateMailbox(cmd.Mailbox)
}

type Delete struct {
	commands.Delete
}

func (cmd *Delete) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	return ctx.User.DeleteMailbox(cmd.Mailbox)
}

type Rename struct {
	commands.Rename
}

func (cmd *Rename) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	return ctx.User.RenameMailbox(cmd.Existing, cmd.New)
}

type Subscribe struct {
	commands.Subscribe
}

func (cmd *Subscribe) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := ctx.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		return err
	}

	return mbox.Subscribe()
}

type Unsubscribe struct {
	commands.Unsubscribe
}

func (cmd *Unsubscribe) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := ctx.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		return err
	}

	return mbox.Unsubscribe()
}

type List struct {
	commands.List
}

func (cmd *List) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	done := make(chan error, 1)

	ch := make(chan *common.MailboxInfo)
	res := &responses.List{Mailboxes: ch, Subscribed: cmd.Subscribed}

	go (func () {
		done <- conn.WriteResp(res)
		close(done)
	})()

	mailboxes, err := ctx.User.ListMailboxes(cmd.Subscribed)
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

		name := info.Name
		if cmd.Reference != "" {
			if !strings.HasSuffix(cmd.Reference, info.Delimiter) {
				cmd.Reference += info.Delimiter
			}
			if !strings.HasPrefix(info.Name, cmd.Reference) {
				continue
			}
			name = strings.TrimPrefix(info.Name, cmd.Reference)
		}

		// TODO: support mixed patterns such as test% or abc*def
		if cmd.Mailbox != name && cmd.Mailbox != "*" && (cmd.Mailbox != "%" || strings.Contains(name, info.Delimiter)) {
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

func (cmd *Status) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := ctx.User.GetMailbox(cmd.Mailbox)
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

func (cmd *Append) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	mbox, err := ctx.User.GetMailbox(cmd.Mailbox)
	if err != nil {
		// TODO: add [TRYCREATE] to the NO response
		return err
	}

	if err := mbox.CreateMessage(cmd.Flags, cmd.Date, cmd.Message.Bytes()); err != nil {
		return err
	}

	// If APPEND targets the currently selected mailbox, send an untagged EXISTS
	// Do this only if the backend doesn't send updates itself
	if conn.Server().Updates == nil && ctx.Mailbox != nil && ctx.Mailbox.Name() == mbox.Name() {
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
