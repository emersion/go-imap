package server

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
)

// imap errors in Authenticated state.
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
		imap.MailboxFlags, imap.MailboxPermanentFlags,
		imap.MailboxMessages, imap.MailboxRecent, imap.MailboxUnseen,
		imap.MailboxUidNext, imap.MailboxUidValidity,
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

	code := imap.CodeReadWrite
	if ctx.MailboxReadOnly {
		code = imap.CodeReadOnly
	}
	return ErrStatusResp(&imap.StatusResp{
		Type: imap.StatusOk,
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

	return mbox.SetSubscribed(true)
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

	return mbox.SetSubscribed(false)
}

type List struct {
	commands.List
}

func (cmd *List) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	ch := make(chan *imap.MailboxInfo)
	res := &responses.List{Mailboxes: ch, Subscribed: cmd.Subscribed}

	done := make(chan error, 1)
	go (func() {
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

		// An empty ("" string) mailbox name argument is a special request to return
		// the hierarchy delimiter and the root name of the name given in the
		// reference.
		if cmd.Mailbox == "" {
			ch <- &imap.MailboxInfo{
				Attributes: []string{imap.NoSelectAttr},
				Delimiter:  info.Delimiter,
				Name:       info.Delimiter,
			}
			break
		}

		if info.Match(cmd.Reference, cmd.Mailbox) {
			ch <- info
		}
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

	// Only keep items thqat have been requested
	items := make(map[string]interface{})
	for _, k := range cmd.Items {
		items[k] = status.Items[k]
	}
	status.Items = items

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
	if err == backend.ErrNoSuchMailbox {
		return ErrStatusResp(&imap.StatusResp{
			Type: imap.StatusNo,
			Code: imap.CodeTryCreate,
			Info: err.Error(),
		})
	} else if err != nil {
		return err
	}

	if err := mbox.CreateMessage(cmd.Flags, cmd.Date, cmd.Message); err != nil {
		return err
	}

	// If APPEND targets the currently selected mailbox, send an untagged EXISTS
	// Do this only if the backend doesn't send updates itself
	if conn.Server().Updates == nil && ctx.Mailbox != nil && ctx.Mailbox.Name() == mbox.Name() {
		status, err := mbox.Status([]string{imap.MailboxMessages})
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
