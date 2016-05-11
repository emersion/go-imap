package server

import (
	"errors"
	"strings"

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

type Check struct {
	commands.Check
}

func (cmd *Check) Handle(conn *Conn) error {
	if conn.Mailbox == nil {
		return errors.New("No mailbox selected")
	}

	return conn.Mailbox.Check()
}

type Close struct {
	commands.Close
}

func (cmd *Close) Handle(conn *Conn) error {
	if conn.Mailbox == nil {
		return errors.New("No mailbox selected")
	}

	if err := conn.Mailbox.Expunge(); err != nil {
		return err
	}

	conn.Mailbox = nil
	conn.MailboxReadOnly = false
	return nil
}

type Expunge struct {
	commands.Expunge
}

func (cmd *Expunge) Handle(conn *Conn) error {
	if conn.Mailbox == nil {
		return errors.New("No mailbox selected")
	}

	// Get a list of messages that will be deleted
	seqids, err := conn.Mailbox.SearchMessages(false, &common.SearchCriteria{Deleted: true})
	if err != nil {
		return err
	}

	if err := conn.Mailbox.Expunge(); err != nil {
		return err
	}

	done := make(chan error)
	defer close(done)

	ch := make(chan uint32)
	res := responses.Expunge{SeqIds: ch}

	go (func () {
		done <- res.WriteTo(conn.Writer)
	})()

	// Iterate sequence numbers from the last one to the first one, as deleting
	// messages changes their respective numbers
	for i := len(seqids) - 1; i >= 0; i-- {
		ch <- seqids[i]
	}
	close(ch)

	return <-done
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

type Store struct {
	commands.Store
}

func (cmd *Store) Handle(conn *Conn) error {
	if conn.Mailbox == nil {
		return errors.New("No mailbox selected")
	}

	itemStr := cmd.Item
	silent := strings.HasSuffix(itemStr, common.SilentOp)
	if silent {
		itemStr = strings.TrimSuffix(itemStr, common.SilentOp)
	}
	item := common.FlagsOp(itemStr)

	if item != common.SetFlags && item != common.AddFlags && item != common.RemoveFlags {
		return errors.New("Unsupported STORE operation")
	}

	flagsList, ok := cmd.Value.([]interface{})
	if !ok {
		return errors.New("Flags must be a list")
	}
	flags, err := common.ParseStringList(flagsList)
	if err != nil {
		return err
	}

	if err := conn.Mailbox.UpdateMessagesFlags(false, cmd.SeqSet, item, flags); err != nil {
		return err
	}

	if !silent { // Not silent: send FETCH updates
		inner := &Fetch{}
		inner.SeqSet = cmd.SeqSet
		inner.Items = []string{"FLAGS"}

		if err := inner.handle(false, conn); err != nil {
			return err
		}
	}

	return nil
}

type Copy struct {
	commands.Copy
}

func (cmd *Copy) handle(uid bool, conn *Conn) error {
	if conn.Mailbox == nil {
		return errors.New("No mailbox selected")
	}

	return conn.Mailbox.CopyMessages(uid, cmd.SeqSet, cmd.Mailbox)
}

func (cmd *Copy) Handle(conn *Conn) error {
	return cmd.handle(false, conn)
}

func (cmd *Copy) UidHandle(conn *Conn) error {
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
