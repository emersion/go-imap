package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

type UserACL struct {
	Email  string
	Rights []imap.Right
}

func (c *Conn) handleGetACL(dec *imapwire.Decoder) error {
	acl, ok := c.session.(SessionACL)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown command",
		}
	}

	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	usersACLs, err := acl.GetACL(mailbox)
	if err != nil {
		return err
	}

	return c.writeUsersACLs(mailbox, usersACLs)
}

func (c *Conn) handleSetACL(dec *imapwire.Decoder) error {
	acl, ok := c.session.(SessionACL)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown command",
		}
	}

	var mailbox string
	var identifier string
	var rights string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectSP() || dec.ExpectAtom(&identifier) || !dec.ExpectSP() || dec.ExpectAtom(&rights) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	parsedRights := parseRights(rights)

	return acl.SetACL(mailbox, identifier, parsedRights)
}

func (c *Conn) handleDeleteACL(dec *imapwire.Decoder) error {
	acl, ok := c.session.(SessionACL)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown command",
		}
	}

	var mailbox string
	var identifier string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectSP() || dec.ExpectAtom(&identifier) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	return acl.DeleteACL(mailbox, identifier)
}

func (c *Conn) writeUsersACLs(mailbox string, usersACLs []*UserACL) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("ACL").SP()
	enc.Atom(mailbox)
	for _, userACL := range usersACLs {
		enc.SP().Atom(userACL.Email).SP()
		for _, r := range userACL.Rights {
			enc.Text(string(r))
		}
	}
	return enc.CRLF()
}

func parseRights(rights string) []imap.Right {
	var rs []imap.Right

	for _, r := range rights {
		right := imap.Right(r)
		rs = append(rs, right)
	}

	return rs
}
