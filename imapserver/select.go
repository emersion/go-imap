package imapserver

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleSelect(tag string, dec *imapwire.Decoder, readOnly bool) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	if c.state == imap.ConnStateSelected {
		if err := c.session.Unselect(); err != nil {
			return err
		}
		c.state = imap.ConnStateAuthenticated
		err := c.writeStatusResp("", &imap.StatusResponse{
			Type: imap.StatusResponseTypeOK,
			Code: "CLOSED",
			Text: "Previous mailbox is now closed",
		})
		if err != nil {
			return err
		}
	}

	options := imap.SelectOptions{ReadOnly: readOnly}
	data, err := c.session.Select(mailbox, &options)
	if err != nil {
		return err
	}

	if err := c.writeExists(data.NumMessages); err != nil {
		return err
	}
	if !c.enabled.Has(imap.CapIMAP4rev2) {
		if err := c.writeObsoleteRecent(); err != nil {
			return err
		}
	}
	if err := c.writeUIDValidity(data.UIDValidity); err != nil {
		return err
	}
	if err := c.writeUIDNext(data.UIDNext); err != nil {
		return err
	}
	if err := c.writeFlags(data.Flags); err != nil {
		return err
	}
	if err := c.writePermanentFlags(data.PermanentFlags); err != nil {
		return err
	}
	if data.List != nil {
		if err := c.writeList(data.List); err != nil {
			return err
		}
	}

	c.state = imap.ConnStateSelected
	// TODO: forbid write commands in read-only mode

	var (
		cmdName string
		code    imap.ResponseCode
	)
	if readOnly {
		cmdName = "EXAMINE"
		code = "READ-ONLY"
	} else {
		cmdName = "SELECT"
		code = "READ-WRITE"
	}
	return c.writeStatusResp(tag, &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Code: code,
		Text: fmt.Sprintf("%v completed", cmdName),
	})
}

func (c *Conn) handleUnselect(dec *imapwire.Decoder, expunge bool) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	if expunge {
		w := &ExpungeWriter{}
		if err := c.session.Expunge(w, nil); err != nil {
			return err
		}
	}

	if err := c.session.Unselect(); err != nil {
		return err
	}

	c.state = imap.ConnStateAuthenticated
	return nil
}

func (c *Conn) writeExists(numMessages uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return enc.Atom("*").SP().Number(numMessages).SP().Atom("EXISTS").CRLF()
}

func (c *Conn) writeObsoleteRecent() error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return enc.Atom("*").SP().Number(0).SP().Atom("RECENT").CRLF()
}

func (c *Conn) writeUIDValidity(uidValidity uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UIDVALIDITY").SP().Number(uidValidity).Special(']')
	enc.SP().Text("UIDs valid")
	return enc.CRLF()
}

func (c *Conn) writeUIDNext(uidNext imap.UID) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UIDNEXT").SP().UID(uidNext).Special(']')
	enc.SP().Text("Predicted next UID")
	return enc.CRLF()
}

func (c *Conn) writeFlags(flags []imap.Flag) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("FLAGS").SP().List(len(flags), func(i int) {
		enc.Flag(flags[i])
	})
	return enc.CRLF()
}

func (c *Conn) writePermanentFlags(flags []imap.Flag) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("PERMANENTFLAGS").SP().List(len(flags), func(i int) {
		enc.Flag(flags[i])
	}).Special(']')
	enc.SP().Text("Permanent flags")
	return enc.CRLF()
}
