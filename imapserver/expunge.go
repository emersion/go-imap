package imapserver

import (
	"fmt"

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
	if err := c.staticSeqSet(seqSet, NumKindUID); err != nil {
		return err
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
	var err error
	c.mutex.Lock()
	if c.state != imap.ConnStateSelected {
		err = fmt.Errorf("imapserver: attempted to write EXPUNGE for sequence number %v but the connection is in the %v state", seqNum, c.state)
	} else if c.numMessages == 0 {
		err = fmt.Errorf("imapserver: attempted to write EXPUNGE for sequence number %v but the selected mailbox is empty", seqNum)
	} else {
		c.numMessages--
	}
	c.mutex.Unlock()
	if err != nil {
		return err
	}

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
