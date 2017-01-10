package server

import (
	"errors"
	"strings"

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

	if err := mailbox.Expunge(); err != nil {
		return err
	}

	// No need to send expunge updates here, since the mailbox is already unselected
	return nil
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

	// Get a list of messages that will be deleted
	// That will allow us to send expunge updates if the backend doesn't support it
	var seqnums []uint32
	if conn.Server().Updates == nil {
		criteria := &imap.SearchCriteria{
			WithFlags: []string{imap.DeletedFlag},
		}

		var err error
		seqnums, err = ctx.Mailbox.SearchMessages(false, criteria)
		if err != nil {
			return err
		}
	}

	if err := ctx.Mailbox.Expunge(); err != nil {
		return err
	}

	// If the backend doesn't support expunge updates, let's do it ourselves
	if conn.Server().Updates == nil {
		done := make(chan error)
		defer close(done)

		ch := make(chan uint32)
		res := &responses.Expunge{SeqNums: ch}

		go (func() {
			done <- conn.WriteResp(res)
		})()

		// Iterate sequence numbers from the last one to the first one, as deleting
		// messages changes their respective numbers
		for i := len(seqnums) - 1; i >= 0; i-- {
			ch <- seqnums[i]
		}
		close(ch)

		if err := <-done; err != nil {
			return err
		}
	}

	return nil
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

	itemStr := cmd.Item
	silent := strings.HasSuffix(itemStr, imap.SilentOp)
	if silent {
		itemStr = strings.TrimSuffix(itemStr, imap.SilentOp)
	}
	item := imap.FlagsOp(itemStr)

	if item != imap.SetFlags && item != imap.AddFlags && item != imap.RemoveFlags {
		return errors.New("Unsupported STORE operation")
	}

	flagsList, ok := cmd.Value.([]interface{})
	if !ok {
		return errors.New("Flags must be a list")
	}
	flags, err := imap.ParseStringList(flagsList)
	if err != nil {
		return err
	}
	for i, flag := range flags {
		flags[i] = imap.CanonicalFlag(flag)
	}

	// If the backend supports message updates, this will prevent this connection
	// from receiving them
	// TODO: find a better way to do this, without conn.silent
	*conn.silent() = silent
	err = ctx.Mailbox.UpdateMessagesFlags(uid, cmd.SeqSet, item, flags)
	*conn.silent() = false
	if err != nil {
		return err
	}

	// Not silent: send FETCH updates if the backend doesn't support message
	// updates
	if conn.Server().Updates == nil && !silent {
		inner := &Fetch{}
		inner.SeqSet = cmd.SeqSet
		inner.Items = []string{"FLAGS"}
		if uid {
			inner.Items = append(inner.Items, "UID")
		}

		if err := inner.handle(uid, conn); err != nil {
			return err
		}
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
		Type: imap.StatusOk,
		Info: imap.Uid + " " + inner.Name + " completed",
	})
}
