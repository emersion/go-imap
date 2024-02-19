package imapserver

import (
	"github.com/opsxolc/go-imap/v2/internal/imapwire"
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
