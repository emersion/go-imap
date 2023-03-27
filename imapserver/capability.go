package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) handleCapability(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	caps := []imap.Cap{imap.CapIMAP4rev2}
	if c.canStartTLS() {
		caps = append(caps, imap.CapStartTLS)
	}
	if c.canAuth() {
		caps = append(caps, imap.Cap("AUTH=PLAIN"))
	} else if c.state == imap.ConnStateNotAuthenticated {
		caps = append(caps, imap.CapLoginDisabled)
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("CAPABILITY")
	for _, c := range caps {
		enc.SP().Atom(string(c))
	}
	return enc.CRLF()
}
