package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleEnable(dec *imapwire.Decoder) error {
	var requested []imap.Cap
	for dec.SP() {
		cap, err := internal.ExpectCap(dec)
		if err != nil {
			return err
		}
		requested = append(requested, cap)
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	available := c.server.options.caps()
	var enabled []imap.Cap
	for _, req := range requested {
		switch req {
		case imap.CapIMAP4rev2, imap.CapUTF8Accept:
			enabled = append(enabled, req)
		case imap.CapCondStore:
			// RFC 7162 §3.6: ENABLE CONDSTORE is a separate
			// activation; only honoured when the backend supports
			// it.
			if available.Has(imap.CapCondStore) {
				enabled = append(enabled, req)
			}
		case imap.CapQResync:
			// RFC 7162 §3.7: enabling QRESYNC implicitly enables
			// CONDSTORE. Both must be backend-supported.
			if available.Has(imap.CapQResync) {
				enabled = append(enabled, req)
				if available.Has(imap.CapCondStore) {
					enabled = append(enabled, imap.CapCondStore)
				}
			}
		}
	}

	c.mutex.Lock()
	for _, e := range enabled {
		c.enabled[e] = struct{}{}
	}
	c.mutex.Unlock()

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("ENABLED")
	for _, c := range enabled {
		enc.SP().Atom(string(c))
	}
	return enc.CRLF()
}
