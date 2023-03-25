package imapserver

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) handleFetch(dec *imapwire.Decoder) error {
	var seqSetStr string
	if !dec.ExpectSP() || !dec.ExpectAtom(&seqSetStr) || !dec.ExpectSP() {
		return dec.Err()
	}
	seqSet, err := imap.ParseSeqSet(seqSetStr)
	if err != nil {
		return err
	}

	var items []imap.FetchItem
	isList, err := dec.List(func() error {
		item, err := readFetchAtt(dec)
		if err != nil {
			return err
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return err
	}
	if !isList {
		item, err := readFetchAtt(dec)
		if err != nil {
			return err
		}
		items = append(items, item)
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	w := &FetchWriter{conn: c}
	if err := c.session.Fetch(w, seqSet, items); err != nil {
		return err
	}
	return nil
}

func readFetchAtt(dec *imapwire.Decoder) (imap.FetchItem, error) {
	var attName string
	if !dec.Expect(dec.Func(&attName, isMsgAttNameChar), "msg-att name") {
		return nil, dec.Err()
	}

	switch attName := imap.FetchItemKeyword(attName); attName {
	case imap.FetchItemAll, imap.FetchItemFast, imap.FetchItemFull:
		return attName, nil
	case imap.FetchItemBodyStructure, imap.FetchItemEnvelope, imap.FetchItemFlags, imap.FetchItemInternalDate, imap.FetchItemRFC822Size, imap.FetchItemUID:
		return attName, nil
	case "BINARY", "BINARY.PEEK":
		part, err := readSectionBinary(dec)
		if err != nil {
			return nil, err
		}
		partial, err := maybeReadPartial(dec)
		if err != nil {
			return nil, err
		}
		return &imap.FetchItemBinarySection{
			Part:    part,
			Partial: partial,
			Peek:    attName == "BINARY.PEEK",
		}, nil
	case "BINARY.SIZE":
		part, err := readSectionBinary(dec)
		if err != nil {
			return nil, err
		}
		return &imap.FetchItemBinarySectionSize{Part: part}, nil
	case "BODY":
		if !dec.Special('[') {
			return attName, nil
		}
		var section imap.FetchItemBodySection
		err := readSection(dec, &section)
		if err != nil {
			return nil, err
		}
		section.Partial, err = maybeReadPartial(dec)
		if err != nil {
			return nil, err
		}
		return &imap.FetchItemBodySection{}, nil
	case "BODY.PEEK":
		if !dec.ExpectSpecial('[') {
			return nil, dec.Err()
		}
		var section imap.FetchItemBodySection
		err := readSection(dec, &section)
		if err != nil {
			return nil, err
		}
		section.Partial, err = maybeReadPartial(dec)
		if err != nil {
			return nil, err
		}
		return &imap.FetchItemBodySection{
			Peek: true,
		}, nil
	default:
		return nil, newClientBugError("Invalid FETCH data item")
	}
}

func isMsgAttNameChar(ch byte) bool {
	return ch != '[' && imapwire.IsAtomChar(ch)
}

func readSection(dec *imapwire.Decoder, section *imap.FetchItemBodySection) error {
	if dec.Special(']') {
		return nil
	}

	var dot bool
	section.Part, dot = readSectionPart(dec)
	if dot {
		var specifier string
		if !dec.ExpectAtom(&specifier) {
			return dec.Err()
		}
		section.Specifier = imap.PartSpecifier(specifier)

		if specifier == "HEADER.FIELDS" || specifier == "HEADER.FIELDS.NOT" {
			if !dec.ExpectSP() {
				return dec.Err()
			}
			var err error
			headerList, err := readHeaderList(dec)
			if err != nil {
				return err
			}
			if specifier == "HEADER.FIELDS" {
				section.HeaderFields = headerList
			} else {
				section.HeaderFieldsNot = headerList
			}
		}
	}

	if !dec.ExpectSpecial(']') {
		return dec.Err()
	}

	return nil
}

func readSectionPart(dec *imapwire.Decoder) (part []int, dot bool) {
	for {
		if len(part) > 0 && !dec.Special('.') {
			return part, false
		}

		var num uint32
		if !dec.Number(&num) {
			return part, true
		}
		part = append(part, int(num))
	}
}

func readHeaderList(dec *imapwire.Decoder) ([]string, error) {
	var l []string
	err := dec.ExpectList(func() error {
		var s string
		if !dec.ExpectAString(&s) {
			return dec.Err()
		}
		l = append(l, s)
		return nil
	})
	return l, err
}

func readSectionBinary(dec *imapwire.Decoder) ([]int, error) {
	if !dec.ExpectSpecial('[') {
		return nil, dec.Err()
	}
	if dec.Special(']') {
		return nil, nil
	}

	var l []int
	for {
		var num uint32
		if !dec.ExpectNumber(&num) {
			return l, dec.Err()
		}
		l = append(l, int(num))

		if !dec.Special('.') {
			break
		}
	}

	if !dec.ExpectSpecial(']') {
		return l, dec.Err()
	}
	return l, nil
}

func maybeReadPartial(dec *imapwire.Decoder) (*imap.SectionPartial, error) {
	if !dec.Special('<') {
		return nil, nil
	}
	var partial imap.SectionPartial
	if !dec.ExpectSpecial('<') || !dec.ExpectNumber64(&partial.Offset) || !dec.ExpectSpecial('.') || !dec.ExpectNumber64(&partial.Size) || !dec.ExpectSpecial('>') {
		return nil, dec.Err()
	}
	return &partial, nil
}

// FetchWriter writes FETCH responses.
type FetchWriter struct {
	conn *conn
}

// CreateMessage writes a FETCH response for a message.
//
// FetchResponseWriter.Close must be called.
func (cmd *FetchWriter) CreateMessage(seqNum uint32) *FetchResponseWriter {
	enc := newResponseEncoder(cmd.conn)
	enc.Atom("*").SP().Number(seqNum).SP().Atom("FETCH").SP().Special('(')
	return &FetchResponseWriter{enc: enc}
}

// FetchResponseWriter writes a single FETCH response for a message.
type FetchResponseWriter struct {
	enc     *responseEncoder
	hasItem bool
}

func (w *FetchResponseWriter) writeItemSep() {
	if w.hasItem {
		w.enc.SP()
	}
	w.hasItem = true
}

// WriteUID writes the message's UID.
func (w *FetchResponseWriter) WriteUID(uid uint32) {
	w.writeItemSep()
	w.enc.Atom("UID").SP().Number(uid)
}

// WriteFlags writes the message's flags.
func (w *FetchResponseWriter) WriteFlags(flags []imap.Flag) {
	w.writeItemSep()
	w.enc.Atom("FLAGS").SP().List(len(flags), func(i int) {
		w.enc.Atom(string(flags[i])) // TODO: validate flag
	})
}

// WriteRFC822Size writes the message's full size.
func (w *FetchResponseWriter) WriteRFC822Size(size int64) {
	w.writeItemSep()
	w.enc.Atom("RFC822.SIZE").SP().Number64(size)
}

// WriteInternalDate writes the message's internal date.
func (w *FetchResponseWriter) WriteInternalDate(t time.Time) {
	w.writeItemSep()
	w.enc.Atom("INTERNALDATE").SP().String(t.Format(internal.DateTimeLayout))
}

// Close closes the FETCH message writer.
func (w *FetchResponseWriter) Close() error {
	if w.enc == nil {
		return fmt.Errorf("imapserver: FetchResponseWriter already closed")
	}
	err := w.enc.Special(')').CRLF()
	w.enc.end()
	w.enc = nil
	return err
}
