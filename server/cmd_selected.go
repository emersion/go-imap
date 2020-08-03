package server

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
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

	return ctx.Mailbox.Poll(true)
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
	_, err := mailbox.Expunge(nil)
	return err
}

type Expunge struct {
	commands.Expunge
}

func (cmd *Expunge) Handle(conn Conn) error {
	if cmd.SeqSet != nil {
		return errors.New("Unexpected argment")
	}

	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}
	if ctx.MailboxReadOnly {
		return ErrMailboxReadOnly
	}

	_, err := ctx.Mailbox.Expunge(nil)
	return err
}

func (cmd *Expunge) UidHandle(conn Conn) error {
	if _, ok := conn.Server().backendExts[backend.ExtUIDPLUS]; !ok {
		return errors.New("Unknown command")
	}
	if cmd.SeqSet == nil {
		return errors.New("Missing set argment")
	}

	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}
	if ctx.MailboxReadOnly {
		return ErrMailboxReadOnly
	}

	_, err := ctx.Mailbox.Expunge([]backend.ExtensionOption{
		backend.ExpungeSeqSet{SeqSet: cmd.SeqSet},
	})
	return err
}

type Search struct {
	commands.Search
}

func (cmd *Search) handle(uid bool, conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox == nil {
		return ErrNoMailboxSelected
	}

	_, ids, err := ctx.Mailbox.SearchMessages(uid, cmd.Criteria, nil)
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

	_, err := ctx.Mailbox.ListMessages(uid, cmd.SeqSet, cmd.Items, ch, nil)
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

	_, err = ctx.Mailbox.UpdateMessagesFlags(uid, cmd.SeqSet, op, silent, flags, nil)
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

	resp, err := ctx.Mailbox.CopyMessages(uid, cmd.SeqSet, cmd.Mailbox, nil)
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

	var customResp *imap.StatusResp
	for _, value := range resp {
		switch value := value.(type) {
		case backend.CopyUIDs:
			customResp = &imap.StatusResp{
				Type: imap.StatusRespOk,
				Code: "COPYUID",
				Arguments: []interface{}{
					value.UIDValidity,
					value.Source,
					value.Dest,
				},
				Info: "COPY completed",
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
