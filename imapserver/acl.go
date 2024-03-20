package imapserver

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleMyRights(dec *imapwire.Decoder) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	session, ok := c.session.(SessionACL)
	if !ok {
		return newClientBugError("MYRIGHTS is not supported")
	}

	data, err := session.MyRights(mailbox)
	if err != nil {
		return err
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("MYRIGHTS").SP().String(data.Mailbox).SP().String(string(data.Rights))
	return enc.CRLF()
}

func (c *Conn) handleSetACL(dec *imapwire.Decoder) error {
	var mailbox, identifier, rights string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectSP() || !dec.ExpectAString(&identifier) ||
		!dec.ExpectSP() || !dec.ExpectAString(&rights) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	session, ok := c.session.(SessionACL)
	if !ok {
		return newClientBugError("SetACL is not supported")
	}

	rm, rs, err := imap.NewRights(rights, false)
	if err != nil {
		return fmt.Errorf("parsing rights error: %v", err)
	}

	return session.SetACL(mailbox, imap.RightsIdentifier(identifier), rm, rs)
}

func (c *Conn) handleGetACL(dec *imapwire.Decoder) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	session, ok := c.session.(SessionACL)
	if !ok {
		return newClientBugError("GETACL is not supported")
	}

	data, err := session.GetACL(mailbox)
	if err != nil {
		return err
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("ACL").SP().String(data.Mailbox)
	for ri, rs := range data.Rights {
		enc.SP().String(string(ri)).SP().String(string(rs))
	}
	return enc.CRLF()
}
