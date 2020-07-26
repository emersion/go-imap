package server

import (
	"bufio"
	"errors"
	"strings"

	"github.com/emersion/go-imap"
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

	// As per RFC1730#6.3.1,
	// 		The SELECT command automatically deselects any
	// 		currently selected mailbox before attempting the new selection.
	// 		Consequently, if a mailbox is selected and a SELECT command that
	// 		fails is attempted, no mailbox is selected.
	// For example, some clients (e.g. Apple Mail) perform SELECT "" when the
	// server doesn't announce the UNSELECT capability.
	ctx.Mailbox = nil
	ctx.MailboxReadOnly = false

	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	status, mbox, err := ctx.User.GetMailbox(cmd.Mailbox, cmd.ReadOnly, conn)
	if err != nil {
		return err
	}

	if ctx.Mailbox != nil {
		ctx.Mailbox.Close()
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
		Type: imap.StatusRespOk,
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

	return ctx.User.SetSubscribed(cmd.Mailbox, true)
}

type Unsubscribe struct {
	commands.Unsubscribe
}

func (cmd *Unsubscribe) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	return ctx.User.SetSubscribed(cmd.Mailbox, false)
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
		// Make sure to drain the channel.
		for range ch {
		}
	})()

	mboxInfo, err := ctx.User.ListMailboxes(cmd.Subscribed)
	if err != nil {
		// Close channel to signal end of results
		close(ch)
		return err
	}

	for _, info := range mboxInfo {
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
			// Do not take pointer to the loop variable.
			info := info

			ch <- &info
		}
	}
	// Close channel to signal end of results
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

	status, err := ctx.User.Status(cmd.Mailbox, cmd.Items)
	if err != nil {
		return err
	}

	// Only keep items thqat have been requested
	items := make(map[imap.StatusItem]interface{})
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

	if err := ctx.User.CreateMessage(cmd.Mailbox, cmd.Flags, cmd.Date, cmd.Message); err != nil {
		if err == backend.ErrNoSuchMailbox {
			return ErrStatusResp(&imap.StatusResp{
				Type: imap.StatusRespNo,
				Code: imap.CodeTryCreate,
				Info: err.Error(),
			})
		}
		if err == backend.ErrTooBig {
			return ErrStatusResp(&imap.StatusResp{
				Type: imap.StatusRespNo,
				Code: "TOOBIG",
				Info: "Message size exceeding limit",
			})
		}
		return err
	}
	return nil
}

type Unselect struct {
	commands.Unselect
}

func (cmd *Unselect) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}

	ctx.Mailbox = nil
	ctx.MailboxReadOnly = false
	return nil
}

type Idle struct {
	commands.Idle
}

func (cmd *Idle) Handle(conn Conn) error {
	cont := &imap.ContinuationReq{Info: "idling"}
	if err := conn.WriteResp(cont); err != nil {
		return err
	}

	// Wait for DONE
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return err
	}

	if strings.ToUpper(scanner.Text()) != "DONE" {
		return errors.New("Expected DONE")
	}
	return nil
}
