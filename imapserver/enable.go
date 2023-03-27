package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) handleEnable(dec *imapwire.Decoder) error {
	for dec.SP() {
		var c string
		if !dec.ExpectAtom(&c) {
			return dec.Err()
		}
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("ENABLED")
	return enc.CRLF()
}
