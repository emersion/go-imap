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

// availableCaps returns the capabilities supported by the server.
//
// They depend on the connection state.
//
// Some extensions (e.g. SASL-IR, ENABLE) don't require backend support and
// thus are always enabled.
func (c *Conn) availableCaps() []imap.Cap {
	available := c.server.options.caps()

	var caps []imap.Cap
	addAvailableCaps(&caps, available, []imap.Cap{
		imap.CapIMAP4rev2,
		imap.CapIMAP4rev1,
	})
	if len(caps) == 0 {
		panic("imapserver: must support at least IMAP4rev1 or IMAP4rev2")
	}

	if available.Has(imap.CapIMAP4rev1) {
		caps = append(caps, []imap.Cap{
			imap.CapSASLIR,
			imap.CapLiteralMinus,
		}...)
	}
	if c.canStartTLS() {
		caps = append(caps, imap.CapStartTLS)
	}
	if c.canAuth() {
		mechs := []string{"PLAIN"}
		if authSess, ok := c.session.(SessionSASL); ok {
			mechs = authSess.AuthenticateMechanisms()
		}
		for _, mech := range mechs {
			caps = append(caps, imap.Cap("AUTH="+mech))
		}
	} else if c.state == imap.ConnStateNotAuthenticated {
		caps = append(caps, imap.CapLoginDisabled)
	}
	if c.state == imap.ConnStateAuthenticated || c.state == imap.ConnStateSelected {
		if available.Has(imap.CapIMAP4rev1) {
			caps = append(caps, []imap.Cap{
				imap.CapUnselect,
				imap.CapEnable,
				imap.CapIdle,
			}...)
			addAvailableCaps(&caps, available, []imap.Cap{
				imap.CapNamespace,
				imap.CapUIDPlus,
				imap.CapESearch,
				imap.CapSearchRes,
				imap.CapListExtended,
				imap.CapListStatus,
				imap.CapMove,
				imap.CapStatusSize,
			})
		}
		addAvailableCaps(&caps, available, []imap.Cap{
			imap.CapCreateSpecialUse,
			imap.CapLiteralPlus,
			imap.CapUnauthenticate,
		})
	}
	return caps
}

func addAvailableCaps(caps *[]imap.Cap, available imap.CapSet, l []imap.Cap) {
	for _, c := range l {
		if available.Has(c) {
			*caps = append(*caps, c)
		}
	}
}
