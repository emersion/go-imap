package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) handleCopy(dec *imapwire.Decoder, numKind NumKind) error {
	seqSet, dest, err := readCopy(dec)
	if err != nil {
		return err
	}
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}
	// TODO: send back COPYUID
	return c.session.Copy(numKind, seqSet, dest)
}

func (c *conn) handleMove(dec *imapwire.Decoder, numKind NumKind) error {
	seqSet, dest, err := readCopy(dec)
	if err != nil {
		return err
	}
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}
	// TODO: send back COPYUID
	return c.session.Move(numKind, seqSet, dest)
}

func readCopy(dec *imapwire.Decoder) (seqSet imap.SeqSet, dest string, err error) {
	var seqSetStr string
	if !dec.ExpectSP() || !dec.ExpectAtom(&seqSetStr) || !dec.ExpectSP() || !dec.ExpectMailbox(&dest) || !dec.ExpectCRLF() {
		return nil, "", dec.Err()
	}
	seqSet, err = imap.ParseSeqSet(seqSetStr)
	return seqSet, dest, err
}
