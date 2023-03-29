package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleCopy(tag string, dec *imapwire.Decoder, numKind NumKind) error {
	seqSet, dest, err := readCopy(dec)
	if err != nil {
		return err
	}
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}
	data, err := c.session.Copy(numKind, seqSet, dest)
	if err != nil {
		return err
	}
	return c.writeCopyOK(tag, data)
}

func (c *Conn) writeCopyOK(tag string, data *imap.CopyData) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	if tag == "" {
		tag = "*"
	}

	enc.Atom(tag).SP().Atom("OK").SP()
	if data != nil {
		enc.Special('[')
		enc.Atom("COPYUID").SP().Number(data.UIDValidity).SP().Atom(data.SourceUIDs.String()).SP().Atom(data.DestUIDs.String())
		enc.Special(']').SP()
	}
	enc.Text("COPY completed")
	return enc.CRLF()
}

func readCopy(dec *imapwire.Decoder) (seqSet imap.SeqSet, dest string, err error) {
	var seqSetStr string
	if !dec.ExpectSP() || !dec.ExpectAtom(&seqSetStr) || !dec.ExpectSP() || !dec.ExpectMailbox(&dest) || !dec.ExpectCRLF() {
		return nil, "", dec.Err()
	}
	seqSet, err = imap.ParseSeqSet(seqSetStr)
	return seqSet, dest, err
}
