package imapserver

import (
	"github.com/opsxolc/go-imap/v2"
	"github.com/opsxolc/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleEnable(dec *imapwire.Decoder) error {
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
		case imap.CapIMAP4rev2, imap.CapUTF8Accept:
			enabled = append(enabled, req)
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
