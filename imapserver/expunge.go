package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleExpunge(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}
	return c.expunge(nil)
}

func (c *Conn) handleUIDExpunge(dec *imapwire.Decoder) error {
	var seqSet imap.SeqSet
	if !dec.ExpectSP() || !dec.ExpectSeqSet(&seqSet) || !dec.ExpectCRLF() {
		return dec.Err()
	}
	return c.expunge(&seqSet)
}

func (c *Conn) expunge(uids *imap.SeqSet) error {
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}
	w := &ExpungeWriter{conn: c}
	return c.session.Expunge(w, uids)
}

func (c *Conn) writeExpunge(seqNum uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Number(seqNum).SP().Atom("EXPUNGE")
	return enc.CRLF()
}

// ExpungeWriter writes EXPUNGE updates.
type ExpungeWriter struct {
	conn *Conn
}

// WriteExpunge notifies the client that the message with the provided sequence
// number has been deleted.
func (w *ExpungeWriter) WriteExpunge(seqNum uint32) error {
	if w.conn == nil {
		return nil
	}
	return w.conn.writeExpunge(seqNum)
}
