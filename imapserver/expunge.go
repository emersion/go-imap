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
	var uidSet imap.UIDSet
	if !dec.ExpectSP() || !dec.ExpectUIDSet(&uidSet) || !dec.ExpectCRLF() {
		return dec.Err()
	}
	return c.expunge(&uidSet)
}

func (c *Conn) expunge(uids *imap.UIDSet) error {
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

// WriteVanished writes a single "* VANISHED <uids>" line carrying
// every UID in uids (RFC 7162 §3.2). A QRESYNC-enabled session uses
// this in place of per-message WriteExpunge to coalesce the EXPUNGE
// response — clients with large reconciliation sets benefit from the
// single line.
//
// The "(EARLIER)" form, used in QRESYNC SELECT responses, is
// produced by the framework via SelectData.Vanished; sessions don't
// call this method during SELECT.
func (w *ExpungeWriter) WriteVanished(uids imap.UIDSet) error {
	if w.conn == nil || len(uids) == 0 {
		return nil
	}
	enc := newResponseEncoder(w.conn)
	defer enc.end()
	enc.Atom("*").SP().Atom("VANISHED").SP().NumSet(uids)
	return enc.CRLF()
}
