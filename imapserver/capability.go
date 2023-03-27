package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleCapability(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("CAPABILITY")
	for _, c := range c.availableCaps() {
		enc.SP().Atom(string(c))
	}
	return enc.CRLF()
}

func (c *Conn) availableCaps() []imap.Cap {
	caps := []imap.Cap{imap.CapIMAP4rev2}
	if c.canStartTLS() {
		caps = append(caps, imap.CapStartTLS)
	}
	if c.canAuth() {
		caps = append(caps, imap.Cap("AUTH=PLAIN"))
	} else if c.state == imap.ConnStateNotAuthenticated {
		caps = append(caps, imap.CapLoginDisabled)
	}
	return caps
}
