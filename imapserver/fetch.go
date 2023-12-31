package imapserver

import (
	"fmt"
	"io"
	"mime"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const envelopeDateLayout = "Mon, 02 Jan 2006 15:04:05 -0700"

type fetchWriterOptions struct {
	bodyStructure struct {
		extended    bool // BODYSTRUCTURE
		nonExtended bool // BODY
	}
	obsolete map[*imap.FetchItemBodySection]string
}

func (c *Conn) handleFetch(dec *imapwire.Decoder, numKind NumKind) error {
	var numSet imap.NumSet
	if !dec.ExpectSP() || !dec.ExpectNumSet(numKind.wire(), &numSet) || !dec.ExpectSP() {
		return dec.Err()
	}

	var options imap.FetchOptions
	writerOptions := fetchWriterOptions{obsolete: make(map[*imap.FetchItemBodySection]string)}
	isList, err := dec.List(func() error {
		name, err := readFetchAttName(dec)
		if err != nil {
			return err
		}
		switch name {
		case "ALL", "FAST", "FULL":
			return newClientBugError("FETCH macros are not allowed in a list")
		}
		return handleFetchAtt(dec, name, &options, &writerOptions)
	})
	if err != nil {
		return err
	}
	if !isList {
		name, err := readFetchAttName(dec)
		if err != nil {
			return err
		}

		// Handle macros
		switch name {
		case "ALL":
			options.Flags = true
			options.InternalDate = true
			options.RFC822Size = true
			options.Envelope = true
		case "FAST":
			options.Flags = true
			options.InternalDate = true
			options.RFC822Size = true
		case "FULL":
			options.Flags = true
			options.InternalDate = true
			options.RFC822Size = true
			options.Envelope = true
			handleFetchBodyStructure(&options, &writerOptions, false)
		default:
			if err := handleFetchAtt(dec, name, &options, &writerOptions); err != nil {
				return err
			}
		}
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	if numKind == NumKindUID {
		options.UID = true
	}

	w := &FetchWriter{conn: c, options: writerOptions}
	if err := c.session.Fetch(w, numSet, &options); err != nil {
		return err
	}
	return nil
}

func handleFetchAtt(dec *imapwire.Decoder, attName string, options *imap.FetchOptions, writerOptions *fetchWriterOptions) error {
	switch attName {
	case "BODYSTRUCTURE":
		handleFetchBodyStructure(options, writerOptions, true)
	case "ENVELOPE":
		options.Envelope = true
	case "FLAGS":
		options.Flags = true
	case "INTERNALDATE":
		options.InternalDate = true
	case "RFC822.SIZE":
		options.RFC822Size = true
	case "UID":
		options.UID = true
	case "RFC822": // equivalent to BODY[]
		bs := &imap.FetchItemBodySection{}
		writerOptions.obsolete[bs] = attName
		options.BodySection = append(options.BodySection, bs)
	case "RFC822.HEADER": // equivalent to BODY.PEEK[HEADER]
		bs := &imap.FetchItemBodySection{
			Specifier: imap.PartSpecifierHeader,
			Peek:      true,
		}
		writerOptions.obsolete[bs] = attName
		options.BodySection = append(options.BodySection, bs)
	case "RFC822.TEXT": // equivalent to BODY[TEXT]
		bs := &imap.FetchItemBodySection{
			Specifier: imap.PartSpecifierText,
		}
		writerOptions.obsolete[bs] = attName
		options.BodySection = append(options.BodySection, bs)
	case "BINARY", "BINARY.PEEK":
		part, err := readSectionBinary(dec)
		if err != nil {
			return err
		}
		partial, err := maybeReadPartial(dec)
		if err != nil {
			return err
		}
		bs := &imap.FetchItemBinarySection{
			Part:    part,
			Partial: partial,
			Peek:    attName == "BINARY.PEEK",
		}
		options.BinarySection = append(options.BinarySection, bs)
	case "BINARY.SIZE":
		part, err := readSectionBinary(dec)
		if err != nil {
			return err
		}
		bss := &imap.FetchItemBinarySectionSize{Part: part}
		options.BinarySectionSize = append(options.BinarySectionSize, bss)
	case "BODY":
		if !dec.Special('[') {
			handleFetchBodyStructure(options, writerOptions, false)
			return nil
		}
		section := imap.FetchItemBodySection{}
		err := readSection(dec, &section)
		if err != nil {
			return err
		}
		section.Partial, err = maybeReadPartial(dec)
		if err != nil {
			return err
		}
		options.BodySection = append(options.BodySection, &section)
	case "BODY.PEEK":
		if !dec.ExpectSpecial('[') {
			return dec.Err()
		}
		section := imap.FetchItemBodySection{Peek: true}
		err := readSection(dec, &section)
		if err != nil {
			return err
		}
		section.Partial, err = maybeReadPartial(dec)
		if err != nil {
			return err
		}
		options.BodySection = append(options.BodySection, &section)
	default:
		return newClientBugError("Unknown FETCH data item")
	}
	return nil
}

func handleFetchBodyStructure(options *imap.FetchOptions, writerOptions *fetchWriterOptions, extended bool) {
	if options.BodyStructure == nil || extended {
		options.BodyStructure = &imap.FetchItemBodyStructure{Extended: extended}
	}
	if extended {
		writerOptions.bodyStructure.extended = true
	} else {
		writerOptions.bodyStructure.nonExtended = true
	}
}

func readFetchAttName(dec *imapwire.Decoder) (string, error) {
	var attName string
	if !dec.Expect(dec.Func(&attName, isMsgAttNameChar), "msg-att name") {
		return "", dec.Err()
	}
	return strings.ToUpper(attName), nil
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
	if dot || len(section.Part) == 0 {
		var specifier string
		if dot {
			if !dec.ExpectAtom(&specifier) {
				return dec.Err()
			}
		} else {
			dec.Atom(&specifier)
		}

		switch specifier := imap.PartSpecifier(strings.ToUpper(specifier)); specifier {
		case imap.PartSpecifierNone, imap.PartSpecifierHeader, imap.PartSpecifierMIME, imap.PartSpecifierText:
			section.Specifier = specifier
		case "HEADER.FIELDS", "HEADER.FIELDS.NOT":
			if !dec.ExpectSP() {
				return dec.Err()
			}
			var err error
			headerList, err := readHeaderList(dec)
			if err != nil {
				return err
			}
			section.Specifier = imap.PartSpecifierHeader
			if specifier == "HEADER.FIELDS" {
				section.HeaderFields = headerList
			} else {
				section.HeaderFieldsNot = headerList
			}
		default:
			return newClientBugError("unknown body section specifier")
		}
	}

	if !dec.ExpectSpecial(']') {
		return dec.Err()
	}

	return nil
}

func readSectionPart(dec *imapwire.Decoder) (part []int, dot bool) {
	for {
		dot = len(part) > 0
		if dot && !dec.Special('.') {
			return part, false
		}

		var num uint32
		if !dec.Number(&num) {
			return part, dot
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
	if !dec.ExpectNumber64(&partial.Offset) || !dec.ExpectSpecial('.') || !dec.ExpectNumber64(&partial.Size) || !dec.ExpectSpecial('>') {
		return nil, dec.Err()
	}
	return &partial, nil
}

// FetchWriter writes FETCH responses.
type FetchWriter struct {
	conn    *Conn
	options fetchWriterOptions
}

// CreateMessage writes a FETCH response for a message.
//
// FetchResponseWriter.Close must be called.
func (cmd *FetchWriter) CreateMessage(seqNum uint32) *FetchResponseWriter {
	enc := newResponseEncoder(cmd.conn)
	enc.Atom("*").SP().Number(seqNum).SP().Atom("FETCH").SP().Special('(')
	return &FetchResponseWriter{enc: enc, options: cmd.options}
}

// FetchResponseWriter writes a single FETCH response for a message.
type FetchResponseWriter struct {
	enc     *responseEncoder
	options fetchWriterOptions

	hasItem bool
}

func (w *FetchResponseWriter) writeItemSep() {
	if w.hasItem {
		w.enc.SP()
	}
	w.hasItem = true
}

// WriteUID writes the message's UID.
func (w *FetchResponseWriter) WriteUID(uid imap.UID) {
	w.writeItemSep()
	w.enc.Atom("UID").SP().UID(uid)
}

// WriteFlags writes the message's flags.
func (w *FetchResponseWriter) WriteFlags(flags []imap.Flag) {
	w.writeItemSep()
	w.enc.Atom("FLAGS").SP().List(len(flags), func(i int) {
		w.enc.Flag(flags[i])
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

// WriteBodySection writes a body section.
//
// The returned io.WriteCloser must be closed before writing any more message
// data items.
func (w *FetchResponseWriter) WriteBodySection(section *imap.FetchItemBodySection, size int64) io.WriteCloser {
	w.writeItemSep()
	enc := w.enc.Encoder

	if obs, ok := w.options.obsolete[section]; ok {
		enc.Atom(obs)
	} else {
		writeItemBodySection(enc, section)
	}

	enc.SP()
	return w.enc.Literal(size)
}

func writeItemBodySection(enc *imapwire.Encoder, section *imap.FetchItemBodySection) {
	enc.Atom("BODY")
	enc.Special('[')
	writeSectionPart(enc, section.Part)
	if len(section.Part) > 0 && section.Specifier != imap.PartSpecifierNone {
		enc.Special('.')
	}
	if section.Specifier != imap.PartSpecifierNone {
		enc.Atom(string(section.Specifier))

		var headerList []string
		if len(section.HeaderFields) > 0 {
			headerList = section.HeaderFields
			enc.Atom(".FIELDS")
		} else if len(section.HeaderFieldsNot) > 0 {
			headerList = section.HeaderFieldsNot
			enc.Atom(".FIELDS.NOT")
		}

		if len(headerList) > 0 {
			enc.SP().List(len(headerList), func(i int) {
				enc.String(headerList[i])
			})
		}
	}
	enc.Special(']')
	if partial := section.Partial; partial != nil {
		enc.Special('<').Number(uint32(partial.Offset)).Special('>')
	}
}

// WriteBinarySection writes a binary section.
//
// The returned io.WriteCloser must be closed before writing any more message
// data items.
func (w *FetchResponseWriter) WriteBinarySection(section *imap.FetchItemBinarySection, size int64) io.WriteCloser {
	w.writeItemSep()
	enc := w.enc.Encoder

	enc.Atom("BINARY").Special('[')
	writeSectionPart(enc, section.Part)
	enc.Special(']').SP()
	enc.Special('~') // indicates literal8
	return w.enc.Literal(size)
}

// WriteBinarySectionSize writes a binary section size.
func (w *FetchResponseWriter) WriteBinarySectionSize(section *imap.FetchItemBinarySection, size uint32) {
	w.writeItemSep()
	enc := w.enc.Encoder

	enc.Atom("BINARY.SIZE").Special('[')
	writeSectionPart(enc, section.Part)
	enc.Special(']').SP().Number(size)
}

// WriteEnvelope writes the message's envelope.
func (w *FetchResponseWriter) WriteEnvelope(envelope *imap.Envelope) {
	w.writeItemSep()
	enc := w.enc.Encoder
	enc.Atom("ENVELOPE").SP()
	writeEnvelope(enc, envelope)
}

// WriteBodyStructure writes the message's body structure (either BODYSTRUCTURE
// or BODY).
func (w *FetchResponseWriter) WriteBodyStructure(bs imap.BodyStructure) {
	if w.options.bodyStructure.nonExtended {
		w.writeBodyStructure(bs, false)
	}

	if w.options.bodyStructure.extended {
		var isExtended bool
		switch bs := bs.(type) {
		case *imap.BodyStructureSinglePart:
			isExtended = bs.Extended != nil
		case *imap.BodyStructureMultiPart:
			isExtended = bs.Extended != nil
		}
		if !isExtended {
			panic("imapserver: client requested extended body structure but a non-extended one is written back")
		}

		w.writeBodyStructure(bs, true)
	}
}

func (w *FetchResponseWriter) writeBodyStructure(bs imap.BodyStructure, extended bool) {
	item := "BODY"
	if extended {
		item = "BODYSTRUCTURE"
	}

	w.writeItemSep()
	enc := w.enc.Encoder
	enc.Atom(item).SP()
	writeBodyStructure(enc, bs, extended)
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

func writeEnvelope(enc *imapwire.Encoder, envelope *imap.Envelope) {
	if envelope == nil {
		envelope = new(imap.Envelope)
	}

	sender := envelope.Sender
	if sender == nil {
		sender = envelope.From
	}
	replyTo := envelope.ReplyTo
	if replyTo == nil {
		replyTo = envelope.From
	}

	enc.Special('(')
	if envelope.Date.IsZero() {
		enc.NIL()
	} else {
		enc.String(envelope.Date.Format(envelopeDateLayout))
	}
	enc.SP()
	writeNString(enc, mime.QEncoding.Encode("utf-8", envelope.Subject))
	addrs := [][]imap.Address{
		envelope.From,
		sender,
		replyTo,
		envelope.To,
		envelope.Cc,
		envelope.Bcc,
	}
	for _, l := range addrs {
		enc.SP()
		writeAddressList(enc, l)
	}
	enc.SP()
	writeNString(enc, envelope.InReplyTo)
	enc.SP()
	writeNString(enc, envelope.MessageID)
	enc.Special(')')
}

func writeAddressList(enc *imapwire.Encoder, l []imap.Address) {
	if l == nil {
		enc.NIL()
		return
	}

	enc.List(len(l), func(i int) {
		addr := l[i]
		enc.Special('(')
		writeNString(enc, mime.QEncoding.Encode("utf-8", addr.Name))
		enc.SP().NIL().SP()
		writeNString(enc, addr.Mailbox)
		enc.SP()
		writeNString(enc, addr.Host)
		enc.Special(')')
	})
}

func writeNString(enc *imapwire.Encoder, s string) {
	if s == "" {
		enc.NIL()
	} else {
		enc.String(s)
	}
}

func writeSectionPart(enc *imapwire.Encoder, part []int) {
	if len(part) == 0 {
		return
	}

	var l []string
	for _, num := range part {
		l = append(l, fmt.Sprintf("%v", num))
	}
	enc.Atom(strings.Join(l, "."))
}

func writeBodyStructure(enc *imapwire.Encoder, bs imap.BodyStructure, extended bool) {
	enc.Special('(')
	switch bs := bs.(type) {
	case *imap.BodyStructureSinglePart:
		writeBodyType1part(enc, bs, extended)
	case *imap.BodyStructureMultiPart:
		writeBodyTypeMpart(enc, bs, extended)
	default:
		panic(fmt.Errorf("unknown body structure type %T", bs))
	}
	enc.Special(')')
}

func writeBodyType1part(enc *imapwire.Encoder, bs *imap.BodyStructureSinglePart, extended bool) {
	enc.String(bs.Type).SP().String(bs.Subtype).SP()
	writeBodyFldParam(enc, bs.Params)
	enc.SP()
	writeNString(enc, bs.ID)
	enc.SP()
	writeNString(enc, bs.Description)
	enc.SP()
	if bs.Encoding == "" {
		enc.String("7BIT")
	} else {
		enc.String(strings.ToUpper(bs.Encoding))
	}
	enc.SP().Number(bs.Size)

	if msg := bs.MessageRFC822; msg != nil {
		enc.SP()
		writeEnvelope(enc, msg.Envelope)
		enc.SP()
		writeBodyStructure(enc, msg.BodyStructure, extended)
		enc.SP().Number64(msg.NumLines)
	} else if text := bs.Text; text != nil {
		enc.SP().Number64(text.NumLines)
	}

	if !extended {
		return
	}
	ext := bs.Extended

	enc.SP()
	enc.NIL() // MD5
	enc.SP()
	writeBodyFldDsp(enc, ext.Disposition)
	enc.SP()
	writeBodyFldLang(enc, ext.Language)
	enc.SP()
	writeNString(enc, ext.Location)
}

func writeBodyTypeMpart(enc *imapwire.Encoder, bs *imap.BodyStructureMultiPart, extended bool) {
	if len(bs.Children) == 0 {
		panic("imapserver: imap.BodyStructureMultiPart must have at least one child")
	}
	for i, child := range bs.Children {
		if i > 0 {
			enc.SP()
		}
		writeBodyStructure(enc, child, extended)
	}

	enc.SP().String(bs.Subtype)

	if !extended {
		return
	}
	ext := bs.Extended

	enc.SP()
	writeBodyFldParam(enc, ext.Params)
	enc.SP()
	writeBodyFldDsp(enc, ext.Disposition)
	enc.SP()
	writeBodyFldLang(enc, ext.Language)
	enc.SP()
	writeNString(enc, ext.Location)
}

func writeBodyFldParam(enc *imapwire.Encoder, params map[string]string) {
	if params == nil {
		enc.NIL()
		return
	}

	var l []string
	for k := range params {
		l = append(l, k)
	}
	sort.Strings(l)

	enc.List(len(l), func(i int) {
		k := l[i]
		v := params[k]
		enc.String(k).SP().String(v)
	})
}

func writeBodyFldDsp(enc *imapwire.Encoder, disp *imap.BodyStructureDisposition) {
	if disp == nil {
		enc.NIL()
		return
	}

	enc.Special('(').String(disp.Value).SP()
	writeBodyFldParam(enc, disp.Params)
	enc.Special(')')
}

func writeBodyFldLang(enc *imapwire.Encoder, l []string) {
	if l == nil {
		enc.NIL()
	} else {
		enc.List(len(l), func(i int) {
			enc.String(l[i])
		})
	}
}
