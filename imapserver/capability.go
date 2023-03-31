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
	caps := []imap.Cap{imap.CapIMAP4rev2, imap.CapIMAP4rev1}
	caps = append(caps, imap4rev1BaseCaps...)
	if c.canStartTLS() {
		caps = append(caps, imap.CapStartTLS)
	}
	if c.canAuth() {
		caps = append(caps, imap.Cap("AUTH=PLAIN"))
	} else if c.state == imap.ConnStateNotAuthenticated {
		caps = append(caps, imap.CapLoginDisabled)
	}
	if c.state == imap.ConnStateAuthenticated || c.state == imap.ConnStateSelected {
		caps = append(caps, imap4rev1AuthCaps...)
	}
	return caps
}

var imap4rev1BaseCaps = []imap.Cap{
	imap.CapSASLIR,
	imap.CapLiteralMinus,
}

var imap4rev1AuthCaps = []imap.Cap{
	imap.CapNamespace,
	imap.CapUnselect,
	imap.CapUIDPlus,
	imap.CapESearch,
	// TODO: implement imap.CapSearchRes
	imap.CapEnable,
	imap.CapIdle,
	imap.CapListExtended,
	imap.CapListStatus,
	imap.CapMove,
	imap.CapStatusSize,
}
