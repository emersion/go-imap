package server

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
)

// imap errors in Selected state.
var (
	ErrNoMailboxSelected = errors.New("No mailbox selected")
	ErrMailboxReadOnly   = errors.New("Mailbox opened in read-only mode")
)

// A command handler that supports UIDs.
type UidHandler interface {
	Handler

	// Handle this command using UIDs for a given connection.
	UidHandle(conn Conn) error
}

type Check struct {
	commands.Check
}

func (cmd *Check) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}
	if ctx.MailboxReadOnly {
		return ErrMailboxReadOnly
	}

	return ctx.Mailbox.Check()
}

type Close struct {
	commands.Close
}

func (cmd *Close) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}

	mailbox := ctx.Mailbox
	ctx.Mailbox = nil
	ctx.MailboxReadOnly = false

	// No need to send expunge updates here, since the mailbox is already unselected
	return mailbox.Expunge()
}

type Expunge struct {
	commands.Expunge
}

func (cmd *Expunge) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}
	if ctx.MailboxReadOnly {
		return ErrMailboxReadOnly
	}

	return ctx.Mailbox.Expunge()
}

type Search struct {
	commands.Search
}

func (cmd *Search) handle(uid bool, conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}

	ids, err := ctx.Mailbox.SearchMessages(uid, cmd.Criteria)
	if err != nil {
		return err
	}

	res := &responses.Search{Ids: ids}
	return conn.WriteResp(res)
}

func (cmd *Search) Handle(conn Conn) error {
	return cmd.handle(false, conn)
}

func (cmd *Search) UidHandle(conn Conn) error {
	return cmd.handle(true, conn)
}

type Fetch struct {
	commands.Fetch
}

func (cmd *Fetch) handle(uid bool, conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}

	ch := make(chan *imap.Message)
	res := &responses.Fetch{Messages: ch}

	done := make(chan error, 1)
	go (func() {
		done <- conn.WriteResp(res)
		// Make sure to drain the message channel.
		for _ = range ch {
		}
	})()

	err := ctx.Mailbox.ListMessages(uid, cmd.SeqSet, cmd.Items, ch)
	if err != nil {
		return err
	}

	return <-done
}

func (cmd *Fetch) Handle(conn Conn) error {
	return cmd.handle(false, conn)
}

func (cmd *Fetch) UidHandle(conn Conn) error {
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

type Store struct {
	commands.Store
}

func (cmd *Store) handle(uid bool, conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}
	if ctx.MailboxReadOnly {
		return ErrMailboxReadOnly
	}

	// Only flags operations are supported
	op, silent, err := imap.ParseFlagsOp(cmd.Item)
	if err != nil {
		return err
	}

	var flags []string

	if flagsList, ok := cmd.Value.([]interface{}); ok {
		// Parse list of flags
		if strs, err := imap.ParseStringList(flagsList); err == nil {
			flags = strs
		} else {
			return err
		}
	} else {
		// Parse single flag
		if str, err := imap.ParseString(cmd.Value); err == nil {
			flags = []string{str}
		} else {
			return err
		}
	}
	for i, flag := range flags {
		flags[i] = imap.CanonicalFlag(flag)
	}

	err = ctx.Mailbox.UpdateMessagesFlags(uid, cmd.SeqSet, op, silent, flags)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *Store) Handle(conn Conn) error {
	return cmd.handle(false, conn)
}

func (cmd *Store) UidHandle(conn Conn) error {
	return cmd.handle(true, conn)
}

type Copy struct {
	commands.Copy
}

func (cmd *Copy) handle(uid bool, conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}

	return ctx.Mailbox.CopyMessages(uid, cmd.SeqSet, cmd.Mailbox)
}

func (cmd *Copy) Handle(conn Conn) error {
	return cmd.handle(false, conn)
}

func (cmd *Copy) UidHandle(conn Conn) error {
	return cmd.handle(true, conn)
}

type Uid struct {
	commands.Uid
}

func (cmd *Uid) Handle(conn Conn) error {
	inner := cmd.Cmd.Command()
	hdlr, err := conn.commandHandler(inner)
	if err != nil {
		return err
	}

	uidHdlr, ok := hdlr.(UidHandler)
	if !ok {
		return errors.New("Command unsupported with UID")
	}

	if err := uidHdlr.UidHandle(conn); err != nil {
		return err
	}

	return ErrStatusResp(&imap.StatusResp{
		Type: imap.StatusRespOk,
		Info: "UID " + inner.Name + " completed",
	})
}
