package imapclient

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Status sends a STATUS command.
func (c *Client) Status(mailbox string, items []imap.StatusItem) *StatusCommand {
	cmd := &StatusCommand{mailbox: mailbox}
	enc := c.beginCommand("STATUS", cmd)
	enc.SP().Mailbox(mailbox).SP()
	enc.List(len(items), func(i int) {
		enc.Atom(string(items[i]))
	})
	enc.end()
	return cmd
}

func (c *Client) handleStatus() error {
	data, err := readStatus(c.dec)
	if err != nil {
		return fmt.Errorf("in status: %v", err)
	}

	cmd := c.findPendingCmdFunc(func(cmd command) bool {
		switch cmd := cmd.(type) {
		case *StatusCommand:
			return cmd.mailbox == data.Mailbox
		case *ListCommand:
			return cmd.returnStatus && cmd.pendingData != nil && cmd.pendingData.Mailbox == data.Mailbox
		default:
			return false
		}
	})
	switch cmd := cmd.(type) {
	case *StatusCommand:
		cmd.data = *data
	case *ListCommand:
		cmd.pendingData.Status = data
		cmd.mailboxes <- cmd.pendingData
		cmd.pendingData = nil
	}

	return nil
}

// StatusCommand is a STATUS command.
type StatusCommand struct {
	cmd
	mailbox string
	data    imap.StatusData
}

func (cmd *StatusCommand) Wait() (*imap.StatusData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

func readStatus(dec *imapwire.Decoder) (*imap.StatusData, error) {
	var data imap.StatusData

	if !dec.ExpectMailbox(&data.Mailbox) || !dec.ExpectSP() {
		return nil, dec.Err()
	}

	err := dec.ExpectList(func() error {
		if err := readStatusAttVal(dec, &data); err != nil {
			return fmt.Errorf("in status-att-val: %v", dec.Err())
		}
		return nil
	})
	return &data, err
}

func readStatusAttVal(dec *imapwire.Decoder, data *imap.StatusData) error {
	var name string
	if !dec.ExpectAtom(&name) || !dec.ExpectSP() {
		return dec.Err()
	}

	var ok bool
	switch imap.StatusItem(strings.ToUpper(name)) {
	case imap.StatusItemNumMessages:
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumMessages = &num
	case imap.StatusItemUIDNext:
		ok = dec.ExpectNumber(&data.UIDNext)
	case imap.StatusItemUIDValidity:
		ok = dec.ExpectNumber(&data.UIDValidity)
	case imap.StatusItemNumUnseen:
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumUnseen = &num
	case imap.StatusItemNumDeleted:
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumDeleted = &num
	case imap.StatusItemSize:
		var size int64
		ok = dec.ExpectNumber64(&size)
		data.Size = &size
	case imap.StatusItemAppendLimit:
		var num uint32
		if dec.Number(&num) {
			ok = true
		} else {
			ok = dec.ExpectNIL()
			num = ^uint32(0)
		}
		data.AppendLimit = &num
	case imap.StatusItemDeletedStorage:
		var storage int64
		ok = dec.ExpectNumber64(&storage)
		data.DeletedStorage = &storage
	default:
		if !dec.DiscardValue() {
			return dec.Err()
		}
	}
	if !ok {
		return dec.Err()
	}
	return nil
}
