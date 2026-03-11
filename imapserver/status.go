package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleStatus(dec *imapwire.Decoder) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectSP() {
		return dec.Err()
	}

	var options imap.StatusOptions
	err := dec.ExpectList(func() error {
		err := readStatusItem(dec, &options)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if options.NumRecent && !c.server.options.caps().Has(imap.CapIMAP4rev1) {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown STATUS data item",
		}
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	data, err := c.session.Status(mailbox, &options)
	if err != nil {
		return err
	}

	return c.writeStatus(data, &options)
}

func (c *Conn) writeStatus(data *imap.StatusData, options *imap.StatusOptions) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom("*").SP().Atom("STATUS").SP().Mailbox(data.Mailbox).SP()
	listEnc := enc.BeginList()
	if options.NumMessages {
		listEnc.Item().Atom("MESSAGES").SP().Number(*data.NumMessages)
	}
	if options.UIDNext {
		listEnc.Item().Atom("UIDNEXT").SP().UID(data.UIDNext)
	}
	if options.UIDValidity {
		listEnc.Item().Atom("UIDVALIDITY").SP().Number(data.UIDValidity)
	}
	if options.NumUnseen {
		listEnc.Item().Atom("UNSEEN").SP().Number(*data.NumUnseen)
	}
	if options.NumDeleted {
		listEnc.Item().Atom("DELETED").SP().Number(*data.NumDeleted)
	}
	if options.Size {
		listEnc.Item().Atom("SIZE").SP().Number64(*data.Size)
	}
	if options.AppendLimit {
		listEnc.Item().Atom("APPENDLIMIT").SP()
		if data.AppendLimit != nil {
			enc.Number(*data.AppendLimit)
		} else {
			enc.NIL()
		}
	}
	if options.DeletedStorage {
		listEnc.Item().Atom("DELETED-STORAGE").SP().Number64(*data.DeletedStorage)
	}
	if options.NumRecent {
		listEnc.Item().Atom("RECENT").SP().Number(*data.NumRecent)
	}
	listEnc.End()

	return enc.CRLF()
}

func readStatusItem(dec *imapwire.Decoder, options *imap.StatusOptions) error {
	var name string
	if !dec.ExpectAtom(&name) {
		return dec.Err()
	}
	switch strings.ToUpper(name) {
	case "MESSAGES":
		options.NumMessages = true
	case "UIDNEXT":
		options.UIDNext = true
	case "UIDVALIDITY":
		options.UIDValidity = true
	case "UNSEEN":
		options.NumUnseen = true
	case "DELETED":
		options.NumDeleted = true
	case "SIZE":
		options.Size = true
	case "APPENDLIMIT":
		options.AppendLimit = true
	case "DELETED-STORAGE":
		options.DeletedStorage = true
	case "RECENT":
		options.NumRecent = true
	default:
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown STATUS data item",
		}
	}
	return nil
}
