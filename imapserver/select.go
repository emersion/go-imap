package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// SelectOptions contains options for the SELECT or EXAMINE command.
type SelectOptions struct {
	ReadOnly bool
}

func (c *conn) handleSelect(dec *imapwire.Decoder, readOnly bool) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	options := SelectOptions{ReadOnly: readOnly}
	data, err := c.session.Select(mailbox, &options)
	if err != nil {
		return err
	}

	if err := c.writeExists(data.NumMessages); err != nil {
		return err
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

	// TODO: send OK with READ-WRITE/READ-ONLY
	c.state = imap.ConnStateSelected
	return nil
}

func (c *conn) handleUnselect(dec *imapwire.Decoder, expunge bool) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	// TODO: expunge

	if err := c.session.Unselect(); err != nil {
		return err
	}

	c.state = imap.ConnStateAuthenticated
	return nil
}

func (c *conn) writeExists(numMessages uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return enc.Atom("*").SP().Number(numMessages).SP().Atom("EXISTS").CRLF()
}

func (c *conn) writeUIDValidity(uidValidity uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UIDVALIDITY").SP().Number(uidValidity).Special(']')
	enc.SP().Text("UIDs valid")
	return enc.CRLF()
}

func (c *conn) writeUIDNext(uidNext uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UIDNEXT").SP().Number(uidNext).Special(']')
	enc.SP().Text("Predicted next UID")
	return enc.CRLF()
}

func (c *conn) writeFlags(flags []imap.Flag) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("FLAGS").SP().List(len(flags), func(i int) {
		enc.Atom(string(flags[i])) // TODO: validate flag
	})
	return enc.CRLF()
}

func (c *conn) writePermanentFlags(flags []imap.Flag) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("PERMANENTFLAGS").SP().List(len(flags), func(i int) {
		enc.Atom(string(flags[i])) // TODO: validate flag
	}).Special(']')
	enc.SP().Text("Permanent flags")
	return enc.CRLF()
}
