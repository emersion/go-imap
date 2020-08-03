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

	// As per RFC1730#6.3.1,
	// 		The SELECT command automatically deselects any
	// 		currently selected mailbox before attempting the new selection.
	// 		Consequently, if a mailbox is selected and a SELECT command that
	// 		fails is attempted, no mailbox is selected.
	// For example, some clients (e.g. Apple Mail) perform SELECT "" when the
	// server doesn't announce the UNSELECT capability.
	if ctx.Mailbox != nil {
		ctx.Mailbox.Close()
	}
	ctx.Mailbox = nil
	ctx.MailboxReadOnly = false

	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	status, mbox, err := ctx.User.GetMailbox(cmd.Mailbox, cmd.ReadOnly, conn, nil)
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

	_, err := ctx.User.CreateMailbox(cmd.Mailbox, nil)
	return err
}

type Delete struct {
	commands.Delete
}

func (cmd *Delete) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	_, err := ctx.User.DeleteMailbox(cmd.Mailbox, nil)
	return err
}

type Rename struct {
	commands.Rename
}

func (cmd *Rename) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.User == nil {
		return ErrNotAuthenticated
	}

	_, err := ctx.User.RenameMailbox(cmd.Existing, cmd.New, nil)
	return err
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

	mboxInfo, err := ctx.User.ListMailboxes(cmd.Subscribed, nil)
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

	res, err := ctx.User.CreateMessage(cmd.Mailbox, cmd.Flags, cmd.Date, cmd.Message, nil)
	if err != nil {
		if err == backend.ErrNoSuchMailbox {
			return ErrStatusResp(&imap.StatusResp{
				Type: imap.StatusRespNo,
				Code: imap.CodeTryCreate,
				Info: "No such mailbox",
			})
		}
		return err
	}

	// If User.CreateMessage is called the backend has no way of knowing it should
	// send any updates while RFC 3501 says it "SHOULD" send EXISTS. This call
	// requests it to send any relevant updates. It may result in it sending
	// more updates than just EXISTS, in particular we allow EXPUNGE updates.
	if ctx.Mailbox != nil && ctx.Mailbox.Name() == cmd.Mailbox {
		return ctx.Mailbox.Poll(true)
	}

	var customResp *imap.StatusResp
	for _, value := range res {
		switch value := value.(type) {
		case backend.AppendUID:
			customResp = &imap.StatusResp{
				Tag:  "",
				Type: imap.StatusRespOk,
				Code: "APPENDUID",
				Arguments: []interface{}{
					value.UIDValidity,
					value.UID,
				},
				Info: "APPEND completed",
			}
		default:
			conn.Server().ErrorLog.Printf("ExtensionResult of unknown type returned by backend: %T", value)
			// Returning an error here would make it look like the command failed.
		}
	}
	if customResp != nil {
		return &imap.ErrStatusResp{Resp: customResp}
	}

	return nil
}
