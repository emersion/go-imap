package imapserver

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleSelect(tag string, dec *imapwire.Decoder, readOnly bool) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) {
		return dec.Err()
	}
	options := imap.SelectOptions{ReadOnly: readOnly}
	// Optional select-modifiers, RFC 7162 §3.1.8:
	//   select-params = SP "(" select-param *(SP select-param) ")"
	if dec.SP() {
		if err := readSelectModifiers(dec, &options); err != nil {
			return err
		}
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	if c.state == imap.ConnStateSelected {
		if err := c.session.Unselect(); err != nil {
			return err
		}
		c.state = imap.ConnStateAuthenticated
		err := c.writeStatusResp("", &imap.StatusResponse{
			Type: imap.StatusResponseTypeOK,
			Code: "CLOSED",
			Text: "Previous mailbox is now closed",
		})
		if err != nil {
			return err
		}
	}

	data, err := c.session.Select(mailbox, &options)
	if err != nil {
		return err
	}

	if err := c.writeExists(data.NumMessages); err != nil {
		return err
	}
	if !c.enabled.Has(imap.CapIMAP4rev2) && c.server.options.caps().Has(imap.CapIMAP4rev1) {
		if err := c.writeObsoleteRecent(data.NumRecent); err != nil {
			return err
		}
		if data.FirstUnseenSeqNum != 0 {
			if err := c.writeObsoleteUnseen(data.FirstUnseenSeqNum); err != nil {
				return err
			}
		}
	}
	if err := c.writeUIDValidity(data.UIDValidity); err != nil {
		return err
	}
	if err := c.writeUIDNext(data.UIDNext); err != nil {
		return err
	}
	if err := c.writeFlags(data.Flags); err != nil {
		return err
	}
	if err := c.writePermanentFlags(data.PermanentFlags); err != nil {
		return err
	}
	// RFC 7162 §3.1.2.1: HIGHESTMODSEQ is reported as a response-text
	// code on an untagged OK line right after PERMANENTFLAGS.
	if data.HighestModSeq != 0 {
		if err := c.writeHighestModSeq(data.HighestModSeq); err != nil {
			return err
		}
	}
	// QRESYNC SELECT (RFC 7162 §3.2): the server emits one
	// "* VANISHED (EARLIER) <uids>" line carrying every UID that has
	// been expunged since the client's last known mod-sequence.
	if len(data.Vanished) > 0 {
		if err := c.writeVanishedEarlier(data.Vanished); err != nil {
			return err
		}
	}
	if data.List != nil {
		if err := c.writeList(data.List); err != nil {
			return err
		}
	}

	c.state = imap.ConnStateSelected
	// TODO: forbid write commands in read-only mode

	var (
		cmdName string
		code    imap.ResponseCode
	)
	if readOnly {
		cmdName = "EXAMINE"
		code = "READ-ONLY"
	} else {
		cmdName = "SELECT"
		code = "READ-WRITE"
	}
	return c.writeStatusResp(tag, &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Code: code,
		Text: fmt.Sprintf("%v completed", cmdName),
	})
}

func (c *Conn) handleUnselect(dec *imapwire.Decoder, expunge bool) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	if expunge {
		w := &ExpungeWriter{}
		if err := c.session.Expunge(w, nil); err != nil {
			return err
		}
	}

	if err := c.session.Unselect(); err != nil {
		return err
	}

	c.state = imap.ConnStateAuthenticated
	return nil
}

func (c *Conn) writeExists(numMessages uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return enc.Atom("*").SP().Number(numMessages).SP().Atom("EXISTS").CRLF()
}

func (c *Conn) writeObsoleteRecent(n uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return enc.Atom("*").SP().Number(n).SP().Atom("RECENT").CRLF()
}

func (c *Conn) writeObsoleteUnseen(n uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UNSEEN").SP().Number(n).Special(']')
	enc.SP().Text("First unseen message")
	return enc.CRLF()
}

func (c *Conn) writeUIDValidity(uidValidity uint32) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UIDVALIDITY").SP().Number(uidValidity).Special(']')
	enc.SP().Text("UIDs valid")
	return enc.CRLF()
}

func (c *Conn) writeUIDNext(uidNext imap.UID) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UIDNEXT").SP().UID(uidNext).Special(']')
	enc.SP().Text("Predicted next UID")
	return enc.CRLF()
}

func (c *Conn) writeFlags(flags []imap.Flag) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("FLAGS").SP().List(len(flags), func(i int) {
		enc.Flag(flags[i])
	})
	return enc.CRLF()
}

func (c *Conn) writePermanentFlags(flags []imap.Flag) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("PERMANENTFLAGS").SP().List(len(flags), func(i int) {
		enc.Flag(flags[i])
	}).Special(']')
	enc.SP().Text("Permanent flags")
	return enc.CRLF()
}

// writeHighestModSeq writes the HIGHESTMODSEQ response-text-code that
// CONDSTORE-enabled clients use to bootstrap their mod-sequence
// state.
func (c *Conn) writeHighestModSeq(modSeq uint64) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("HIGHESTMODSEQ").SP().ModSeq(modSeq).Special(']')
	enc.SP().Text("Highest")
	return enc.CRLF()
}

// writeVanishedEarlier writes one "* VANISHED (EARLIER) <uids>" line
// listing the UIDs the client has lost (during a QRESYNC SELECT, RFC
// 7162 §3.2).
func (c *Conn) writeVanishedEarlier(uids imap.UIDSet) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("VANISHED").SP()
	enc.Special('(').Atom("EARLIER").Special(')')
	enc.SP().NumSet(uids)
	return enc.CRLF()
}

// readSelectModifiers parses the parenthesised list of select-params
// that may follow the mailbox name. CONDSTORE (RFC 7162 §3.1.8) and
// QRESYNC (§3.2) are supported.
func readSelectModifiers(dec *imapwire.Decoder, options *imap.SelectOptions) error {
	return dec.ExpectList(func() error {
		var name string
		if !dec.ExpectAtom(&name) {
			return dec.Err()
		}
		switch strings.ToUpper(name) {
		case "CONDSTORE":
			options.CondStore = true
		case "QRESYNC":
			q, err := readQResyncParams(dec)
			if err != nil {
				return err
			}
			options.QResync = q
		default:
			return newClientBugError("unknown SELECT modifier")
		}
		return nil
	})
}

// readQResyncParams parses
//
//	QRESYNC "(" uidvalidity SP mod-sequence-value [SP known-uids] ")"
//
// per RFC 7162 §3.2. The optional reconciliation pair
// "(known-sequence-set " known-uid-set ")" the spec also describes is
// an advanced cache optimisation rare in the wild; we accept it as
// well but only expose the known UIDs to the session, leaving the
// seq/uid pairs unused.
func readQResyncParams(dec *imapwire.Decoder) (*imap.QResyncOptions, error) {
	if !dec.ExpectSP() || !dec.ExpectSpecial('(') {
		return nil, dec.Err()
	}
	q := &imap.QResyncOptions{}
	if !dec.ExpectNumber(&q.UIDValidity) || !dec.ExpectSP() || !dec.ExpectModSeq(&q.ModSeq) {
		return nil, dec.Err()
	}
	// Optional known-uids.
	if dec.SP() {
		// Either "(seqset uidset)" reconciliation pair OR plain UID
		// set. Peek the next byte: '(' → pair, anything else → uid
		// set.
		if dec.Special('(') {
			// We accept the reconciliation pair but discard it —
			// honouring it requires the session to know the client's
			// pre-expunge sequence view, which v1 does not track.
			var (
				dummySeq imap.NumSet
				dummyUID imap.UIDSet
			)
			if !dec.ExpectNumSet(imapwire.NumKindSeq, &dummySeq) ||
				!dec.ExpectSP() || !dec.ExpectUIDSet(&dummyUID) || !dec.ExpectSpecial(')') {
				return nil, dec.Err()
			}
		} else {
			if !dec.ExpectUIDSet(&q.KnownUIDs) {
				return nil, dec.Err()
			}
		}
	}
	if !dec.ExpectSpecial(')') {
		return nil, dec.Err()
	}
	return q, nil
}
