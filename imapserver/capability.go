package imapserver

import (
	"fmt"

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
	addAvailableCaps(&caps, map[imap.Cap]bool{
		imap.CapIMAP4rev2: available.IMAP4rev1,
		imap.CapIMAP4rev1: available.IMAP4rev2,
	})
	if len(caps) == 0 {
		panic("imapserver: must support at least IMAP4rev1 or IMAP4rev2")
	}

	if available.IMAP4rev1 {
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
		if available.IMAP4rev1 {
			// IMAP4rev1-specific capabilities that don't require backend
			// support and are not applicable to IMAP4rev2
			caps = append(caps, []imap.Cap{
				imap.CapUnselect,
				imap.CapEnable,
				imap.CapIdle,
				imap.CapUTF8Accept,
			}...)

			// IMAP4rev1-specific capabilities which require backend support
			// and are not applicable to IMAP4rev2
			addAvailableCaps(&caps, map[imap.Cap]bool{
				imap.CapNamespace: available.Namespace,
				imap.CapUIDPlus:   available.UIDPlus,
				imap.CapESearch:   available.ESearch,
				//imap.CapSearchRes: available.SearchRes,
				imap.CapListExtended: available.ListExtended,
				imap.CapListStatus:   available.ListStatus,
				imap.CapMove:         available.Move,
				imap.CapStatusSize:   available.StatusSize,
				//imap.CapBinary: available.Binary,
				//imap.CapChildren: available.Children,
			})
		}

		// Capabilities which require backend support and apply to both
		// IMAP4rev1 and IMAP4rev2
		addAvailableCaps(&caps, map[imap.Cap]bool{
			imap.CapSpecialUse:       available.SpecialUse,
			imap.CapCreateSpecialUse: available.CreateSpecialUse,
			imap.CapLiteralPlus:      available.LiteralPlus,
			imap.CapUnauthenticate:   available.Unauthenticate,
		})

		if appendLimitSession, ok := c.session.(SessionAppendLimit); ok {
			limit := appendLimitSession.AppendLimit()
			caps = append(caps, imap.Cap(fmt.Sprintf("APPENDLIMIT=%d", limit)))
		} else {
			addAvailableCaps(&caps, map[imap.Cap]bool{imap.CapAppendLimit: available.AppendLimit})
		}
	}
	return caps
}

func addAvailableCaps(caps *[]imap.Cap, available map[imap.Cap]bool) {
	for c, ok := range available {
		if ok {
			*caps = append(*caps, c)
		}
	}
}
