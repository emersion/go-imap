package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleMove(dec *imapwire.Decoder, numKind NumKind) error {
	numSet, dest, err := readCopy(numKind, dec)
	if err != nil {
		return err
	}
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}
	session, ok := c.session.(SessionMove)
	if !ok {
		return newClientBugError("MOVE is not supported")
	}
	w := &MoveWriter{conn: c}
	return session.Move(w, numSet, dest)
}

// MoveWriter writes responses for the MOVE command.
//
// Servers must first call WriteCopyData once, then call WriteExpunge any
// number of times.
type MoveWriter struct {
	conn *Conn
}

// WriteCopyData writes the untagged COPYUID response for a MOVE command.
func (w *MoveWriter) WriteCopyData(data *imap.CopyData) error {
	return w.conn.writeCopyOK("", data)
}

// WriteExpunge writes an EXPUNGE response for a MOVE command.
func (w *MoveWriter) WriteExpunge(seqNum uint32) error {
	return w.conn.writeExpunge(seqNum)
}
