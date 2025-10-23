package imapserver

import (
	"github.com/emersion/go-imap/v2"
)

// VanishedWriter writes VANISHED updates for QRESYNC-enabled connections.
type VanishedWriter struct {
	conn *Conn
}

// WriteVanished notifies the client that the messages with the provided UIDs
// have been expunged. If earlier is true, this is a VANISHED (EARLIER) response.
func (w *VanishedWriter) WriteVanished(uids imap.UIDSet, earlier bool) error {
	if w.conn == nil {
		return nil
	}
	return w.conn.writeVanished(uids, earlier)
}

func (c *Conn) writeVanished(uids imap.UIDSet, earlier bool) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("VANISHED")
	if earlier {
		enc.SP().List(1, func(i int) {
			enc.Atom("EARLIER")
		})
	}
	enc.SP().NumSet(uids)
	return enc.CRLF()
}
