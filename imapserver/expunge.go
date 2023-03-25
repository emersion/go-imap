package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) handleExpunge(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}
	return c.expunge(nil)
}

func (c *conn) handleUIDExpunge(dec *imapwire.Decoder) error {
	var seqSetStr string
	if !dec.ExpectSP() || !dec.ExpectAtom(&seqSetStr) || !dec.ExpectCRLF() {
		return dec.Err()
	}
	seqSet, err := imap.ParseSeqSet(seqSetStr)
	if err != nil {
		return err
	}
	return c.expunge(&seqSet)
}

func (c *conn) expunge(uids *imap.SeqSet) error {
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}
	return c.session.Expunge(uids)
}
