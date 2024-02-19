package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleCopy(tag string, dec *imapwire.Decoder, numKind NumKind) error {
	numSet, dest, err := readCopy(numKind, dec)
	if err != nil {
		return err
	}
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}
	data, err := c.session.Copy(numSet, dest)
	if err != nil {
		return err
	}

	cmdName := "COPY"
	if numKind == NumKindUID {
		cmdName = "UID COPY"
	}
	if err := c.poll(cmdName); err != nil {
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
		enc.Atom("COPYUID").SP().Number(data.UIDValidity).SP().NumSet(data.SourceUIDs).SP().NumSet(data.DestUIDs)
		enc.Special(']').SP()
	}
	enc.Text("COPY completed")
	return enc.CRLF()
}

func readCopy(numKind NumKind, dec *imapwire.Decoder) (numSet imap.NumSet, dest string, err error) {
	if !dec.ExpectSP() || !dec.ExpectNumSet(numKind.wire(), &numSet) || !dec.ExpectSP() || !dec.ExpectMailbox(&dest) || !dec.ExpectCRLF() {
		return nil, "", dec.Err()
	}
	return numSet, dest, nil
}
