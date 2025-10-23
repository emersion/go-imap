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

	if dec.SP() {
		err := dec.ExpectList(func() error {
			var param string
			if !dec.ExpectAtom(&param) {
				return dec.Err()
			}

			switch strings.ToUpper(param) {
			case "CONDSTORE":
				// Per RFC 7162, ignore the parameter if not supported.
				if c.server.options.caps().Has(imap.CapCondStore) || c.enabled.Has(imap.CapIMAP4rev2) {
					options.CondStore = true
				}
			case "QRESYNC":
				// Per RFC 7162, QRESYNC requires ENABLE QRESYNC
				if c.enabled.Has(imap.CapQResync) {
					if !dec.ExpectSP() {
						return dec.Err()
					}
					err := dec.ExpectList(func() error {
						var uidValidity uint32
						var modSeq uint64
						if !dec.ExpectNumber(&uidValidity) || !dec.ExpectSP() || !dec.ExpectModSeq(&modSeq) {
							return dec.Err()
						}

						qresyncData := &imap.QResyncData{
							UIDValidity: uidValidity,
							ModSeq:      modSeq,
						}

						if dec.SP() {
							var knownUIDs imap.UIDSet
							if dec.ExpectUIDSet(&knownUIDs) {
								qresyncData.KnownUIDs = knownUIDs

								if dec.SP() {
									// Optional sequence/UID match data
									err := dec.ExpectList(func() error {
										var seqNums, uids imap.UIDSet
										if !dec.ExpectUIDSet(&seqNums) || !dec.ExpectSP() || !dec.ExpectUIDSet(&uids) {
											return dec.Err()
										}
										qresyncData.SeqMatch = &imap.QResyncSeqMatch{
											SeqNums: seqNums,
											UIDs:    uids,
										}
										return nil
									})
									if err != nil {
										return err
									}
								}
							}
						}

						options.QResync = qresyncData
						return nil
					})
					if err != nil {
						return err
					}
				}
			default:
				return newClientBugError(fmt.Sprintf("unknown SELECT parameter: %v", param))
			}
			return nil
		})
		if err != nil {
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

	enc := newResponseEncoder(c)
	defer enc.end()

	isQResync := options.QResync != nil && data.UIDValidity == options.QResync.UIDValidity
	if !isQResync {
		writeExists(enc.Encoder, data.NumMessages)
		if !c.enabled.Has(imap.CapIMAP4rev2) && c.server.options.caps().Has(imap.CapIMAP4rev1) {
			writeObsoleteRecent(enc.Encoder, data.NumRecent)
			if data.FirstUnseenSeqNum != 0 {
				writeObsoleteUnseen(enc.Encoder, data.FirstUnseenSeqNum)
			}
		}
	}

	if len(data.Vanished) > 0 {
		writeVanished(enc.Encoder, data.Vanished)
	}

	if len(data.Modified) > 0 {
		for _, mod := range data.Modified {
			if err := writeQResyncFetch(enc.Encoder, mod); err != nil {
				return err
			}
		}
	}

	writeUIDValidity(enc.Encoder, data.UIDValidity)
	writeUIDNext(enc.Encoder, data.UIDNext)
	writeFlags(enc.Encoder, data.Flags)
	writePermanentFlags(enc.Encoder, data.PermanentFlags)
	if data.List != nil {
		if err := c.writeList(data.List); err != nil {
			return err
		}
	}

	if c.server.options.caps().Has(imap.CapCondStore) {
		if data.HighestModSeq > 0 {
			writeHighestModSeq(enc.Encoder, data.HighestModSeq)
		} else {
			writeNoModSeq(enc.Encoder)
		}
	}

	c.state = imap.ConnStateSelected

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
	return writeStatusResp(enc.Encoder, tag, &imap.StatusResponse{
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
		w := &ExpungeWriter{conn: c}
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

func writeObsoleteUnseen(enc *imapwire.Encoder, n uint32) error {
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UNSEEN").SP().Number(n).Special(']')
	enc.SP().Text("First unseen message")
	return enc.CRLF()
}

func writeUIDValidity(enc *imapwire.Encoder, uidValidity uint32) error {
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UIDVALIDITY").SP().Number(uidValidity).Special(']')
	enc.SP().Text("UIDs valid")
	return enc.CRLF()
}

func writeUIDNext(enc *imapwire.Encoder, uidNext imap.UID) error {
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("UIDNEXT").SP().UID(uidNext).Special(']')
	enc.SP().Text("Predicted next UID")
	return enc.CRLF()
}

func (c *Conn) writeFlags(flags []imap.Flag) error { // also used by UpdateWriter
	enc := newResponseEncoder(c)
	defer enc.end()
	return writeFlags(enc.Encoder, flags)
}

func writeFlags(enc *imapwire.Encoder, flags []imap.Flag) error {
	enc.Atom("*").SP().Atom("FLAGS").SP().List(len(flags), func(i int) {
		enc.Flag(flags[i])
	})
	return enc.CRLF()
}

func writePermanentFlags(enc *imapwire.Encoder, flags []imap.Flag) error {
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("PERMANENTFLAGS").SP().List(len(flags), func(i int) {
		enc.Flag(flags[i])
	}).Special(']')
	enc.SP().Text("Permanent flags")
	return enc.CRLF()
}

func writeHighestModSeq(enc *imapwire.Encoder, highestModSeq uint64) error {
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("HIGHESTMODSEQ").SP().ModSeq(highestModSeq).Special(']')
	enc.SP().Text("Highest modification sequence")
	return enc.CRLF()
}

func writeNoModSeq(enc *imapwire.Encoder) error {
	enc.Atom("*").SP().Atom("OK").SP()
	enc.Special('[').Atom("NOMODSEQ").Special(']')
	enc.SP().Text("Mailbox does not support modification sequences")
	return enc.CRLF()
}

func writeVanished(enc *imapwire.Encoder, uids imap.UIDSet) error {
	enc.Atom("*").SP().Atom("VANISHED").SP()
	enc.NumSet(uids)
	return enc.CRLF()
}

func writeQResyncFetch(enc *imapwire.Encoder, mod imap.SelectModifiedData) error {
	enc.Atom("*").SP().Number(mod.SeqNum).SP().Atom("FETCH").SP().Special('(')
	enc.Atom("UID").SP().UID(mod.UID)
	enc.SP().Atom("FLAGS").SP().List(len(mod.Flags), func(i int) {
		enc.Flag(mod.Flags[i])
	})
	enc.SP().Atom("MODSEQ").SP().ModSeq(mod.ModSeq)
	enc.Special(')')
	return enc.CRLF()
}
