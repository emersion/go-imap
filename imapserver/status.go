package imapserver

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) handleStatus(dec *imapwire.Decoder) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectSP() {
		return dec.Err()
	}

	var items []imap.StatusItem
	err := dec.ExpectList(func() error {
		item, err := readStatusItem(dec)
		if err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return err
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	data, err := c.session.Status(mailbox, items)
	if err != nil {
		return err
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	return enc.Atom("*").SP().Atom("STATUS").SP().Mailbox(data.Mailbox).SP().List(len(items), func(i int) {
		item := items[i]
		enc.Atom(string(item)).SP()
		switch item {
		case imap.StatusItemNumMessages:
			enc.Number(*data.NumMessages)
		case imap.StatusItemUIDNext:
			enc.Number(data.UIDNext)
		case imap.StatusItemUIDValidity:
			enc.Number(data.UIDValidity)
		case imap.StatusItemNumUnseen:
			enc.Number(*data.NumUnseen)
		case imap.StatusItemNumDeleted:
			enc.Number(*data.NumDeleted)
		case imap.StatusItemSize:
			enc.Number64(*data.Size)
		case imap.StatusItemAppendLimit:
			if data.AppendLimit != nil {
				enc.Number(*data.AppendLimit)
			} else {
				enc.NIL()
			}
		case imap.StatusItemDeletedStorage:
			enc.Number64(*data.DeletedStorage)
		default:
			panic(fmt.Errorf("imapserver: unknown STATUS item %v", item))
		}
	}).CRLF()
}

func readStatusItem(dec *imapwire.Decoder) (imap.StatusItem, error) {
	var name string
	if !dec.ExpectAtom(&name) {
		return "", dec.Err()
	}
	switch item := imap.StatusItem(name); item {
	case imap.StatusItemNumMessages, imap.StatusItemUIDNext, imap.StatusItemUIDValidity, imap.StatusItemNumUnseen, imap.StatusItemNumDeleted, imap.StatusItemSize, imap.StatusItemAppendLimit, imap.StatusItemDeletedStorage:
		return item, nil
	default:
		return "", &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown STATUS data item",
		}
	}
}
