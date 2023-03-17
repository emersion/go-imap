package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Status sends a STATUS command.
func (c *Client) Status(mailbox string, items []StatusItem) *StatusCommand {
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
	data    StatusData
}

func (cmd *StatusCommand) Wait() (*StatusData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// StatusItem is a data item which can be requested by a STATUS command.
type StatusItem string

const (
	StatusItemNumMessages StatusItem = "MESSAGES"
	StatusItemUIDNext     StatusItem = "UIDNEXT"
	StatusItemUIDValidity StatusItem = "UIDVALIDITY"
	StatusItemNumUnseen   StatusItem = "UNSEEN"
	StatusItemNumDeleted  StatusItem = "DELETED" // requires IMAP4rev2
	StatusItemSize        StatusItem = "SIZE"    // requires IMAP4rev2 or STATUS=SIZE

	StatusItemAppendLimit StatusItem = "APPENDLIMIT" // requires APPENDLIMIT
)

// StatusData is the data returned by a STATUS command.
//
// The mailbox name is always populated. The remaining fields are optional.
type StatusData struct {
	Mailbox string

	NumMessages *uint32
	UIDNext     uint32
	UIDValidity uint32
	NumUnseen   *uint32
	NumDeleted  *uint32
	Size        *int64

	AppendLimit *uint32
}

func readStatus(dec *imapwire.Decoder) (*StatusData, error) {
	var data StatusData

	var err error
	if data.Mailbox, err = dec.ExpectMailbox(); err != nil {
		return nil, err
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	err = dec.ExpectList(func() error {
		if err := readStatusAttVal(dec, &data); err != nil {
			return fmt.Errorf("in status-att-val: %v", dec.Err())
		}
		return nil
	})
	return &data, err
}

func readStatusAttVal(dec *imapwire.Decoder, data *StatusData) error {
	var name string
	if !dec.ExpectAtom(&name) || !dec.ExpectSP() {
		return dec.Err()
	}

	var ok bool
	switch StatusItem(name) {
	case StatusItemNumMessages:
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumMessages = &num
	case StatusItemUIDNext:
		ok = dec.ExpectNumber(&data.UIDNext)
	case StatusItemUIDValidity:
		ok = dec.ExpectNumber(&data.UIDValidity)
	case StatusItemNumUnseen:
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumUnseen = &num
	case StatusItemNumDeleted:
		var num uint32
		ok = dec.ExpectNumber(&num)
		data.NumDeleted = &num
	case StatusItemSize:
		var size int64
		ok = dec.ExpectNumber64(&size)
		data.Size = &size
	case StatusItemAppendLimit:
		var num uint32
		if dec.Number(&num) {
			ok = true
		} else {
			ok = dec.ExpectNIL()
			num = ^uint32(0)
		}
		data.AppendLimit = &num
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
