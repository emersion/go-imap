package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Status sends a STATUS command.
func (c *Client) Status(mailbox string, items []StatusItem) *StatusCommand {
	cmd := &StatusCommand{}
	enc := c.beginCommand("STATUS", cmd)
	enc.SP().Mailbox(mailbox).SP()
	enc.List(len(items), func(i int) {
		enc.Atom(string(items[i]))
	})
	enc.end()
	return cmd
}

// StatusCommand is a STATUS command.
type StatusCommand struct {
	cmd
	data StatusData
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
	StatusItemNumDeleted  StatusItem = "DELETED"
	StatusItemSize        StatusItem = "SIZE"
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
}

func readStatus(dec *imapwire.Decoder, cmd *StatusCommand) error {
	var data *StatusData
	if cmd != nil {
		data = &cmd.data
	} else {
		data = &StatusData{}
	}

	var err error
	if data.Mailbox, err = dec.ExpectMailbox(); err != nil {
		return err
	}

	if !dec.ExpectSP() {
		return dec.Err()
	}

	return dec.ExpectList(func() error {
		if err := readStatusAttVal(dec, data); err != nil {
			return fmt.Errorf("in status-att-val: %v", dec.Err())
		}
		return nil
	})
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
		num, ok = dec.ExpectNumber()
		data.NumMessages = &num
	case StatusItemUIDNext:
		data.UIDNext, ok = dec.ExpectNumber()
	case StatusItemUIDValidity:
		data.UIDValidity, ok = dec.ExpectNumber()
	case StatusItemNumUnseen:
		var num uint32
		num, ok = dec.ExpectNumber()
		data.NumUnseen = &num
	case StatusItemNumDeleted:
		var num uint32
		num, ok = dec.ExpectNumber()
		data.NumDeleted = &num
	case StatusItemSize:
		var size int64
		size, ok = dec.ExpectNumber64()
		data.Size = &size
	default:
		// TODO: skip tagged-ext
		return fmt.Errorf("unsupported status-att-val %q", name)
	}
	if !ok {
		return dec.Err()
	}
	return nil
}
