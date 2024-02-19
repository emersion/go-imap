package imapserver

import (
	"github.com/opsxolc/go-imap/v2"
	"github.com/opsxolc/go-imap/v2/internal/imapwire"
)

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

	return session.SetACL(mailbox, imap.RightsIdentifier(identifier), rights)
}
