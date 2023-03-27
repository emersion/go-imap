package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) handleEnable(dec *imapwire.Decoder) error {
	var requested []imap.Cap
	for dec.SP() {
		var c string
		if !dec.ExpectAtom(&c) {
			return dec.Err()
		}
		requested = append(requested, imap.Cap(c))
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	var enabled []imap.Cap
	for _, req := range requested {
		switch req {
		case imap.CapIMAP4rev2:
			c.enabled[req] = struct{}{}
			enabled = append(enabled, req)
		}
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("ENABLED")
	for _, c := range enabled {
		enc.SP().Atom(string(c))
	}
	return enc.CRLF()
}
