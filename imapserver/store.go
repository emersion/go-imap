package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleStore(dec *imapwire.Decoder, numKind NumKind) error {
	var seqSetStr, item string
	if !dec.ExpectSP() || !dec.ExpectAtom(&seqSetStr) || !dec.ExpectSP() || !dec.ExpectAtom(&item) || !dec.ExpectSP() {
		return dec.Err()
	}
	flags, err := internal.ReadFlagList(dec)
	if err != nil {
		return err
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	seqSet, err := imap.ParseSeqSet(seqSetStr)
	if err != nil {
		return err
	}

	silent := strings.HasSuffix(item, ".SILENT")
	item = strings.TrimSuffix(item, ".SILENT")

	var op imap.StoreFlagsOp
	switch {
	case strings.HasPrefix(item, "+"):
		op = imap.StoreFlagsAdd
		item = strings.TrimPrefix(item, "+")
	case strings.HasPrefix(item, "-"):
		op = imap.StoreFlagsDel
		item = strings.TrimPrefix(item, "-")
	default:
		op = imap.StoreFlagsSet
	}

	if item != "FLAGS" {
		return newClientBugError("STORE can only change FLAGS")
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	w := &FetchWriter{conn: c}
	return c.session.Store(w, numKind, seqSet, &imap.StoreFlags{
		Op:     op,
		Silent: silent,
		Flags:  flags,
	})
}
