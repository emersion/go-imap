package imapclient

import (
	"fmt"
	"io"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
	"github.com/emersion/go-message/mail"
)

// Fetch sends a FETCH command.
//
// The caller must fully consume the FetchCommand. A simple way to do so is to
// defer a call to FetchCommand.Close.
//
// A nil options pointer is equivalent to a zero options value.
func (c *Client) Fetch(numSet imap.NumSet, options *imap.FetchOptions) *FetchCommand {
	if options == nil {
		options = new(imap.FetchOptions)
	}

	numKind := imapwire.NumSetKind(numSet)

	cmd := &FetchCommand{
		numSet: numSet,
		msgs:   make(chan *FetchMessageData, 128),
	}
	enc := c.beginCommand(uidCmdName("FETCH", numKind), cmd)
	enc.SP().NumSet(numSet).SP()
	writeFetchItems(enc.Encoder, numKind, options)
	if options.ChangedSince != 0 {
		enc.SP().Special('(').Atom("CHANGEDSINCE").SP().ModSeq(options.ChangedSince).Special(')')
	}
	enc.end()
	return cmd
}

func writeFetchItems(enc *imapwire.Encoder, numKind imapwire.NumKind, options *imap.FetchOptions) {
	listEnc := enc.BeginList()

	// Ensure we request UID as the first data item for UID FETCH, to be safer.
	// We want to get it before any literal.
	if options.UID || numKind == imapwire.NumKindUID {
		listEnc.Item().Atom("UID")
	}

	m := map[string]bool{
		"BODY":          options.BodyStructure != nil && !options.BodyStructure.Extended,
		"BODYSTRUCTURE": options.BodyStructure != nil && options.BodyStructure.Extended,
		"ENVELOPE":      options.Envelope,
		"FLAGS":         options.Flags,
		"INTERNALDATE":  options.InternalDate,
		"RFC822.SIZE":   options.RFC822Size,
		"MODSEQ":        options.ModSeq,
	}
	for k, req := range m {
		if req {
			listEnc.Item().Atom(k)
		}
	}

	for _, bs := range options.BodySection {
		writeFetchItemBodySection(listEnc.Item(), bs)
	}
	for _, bs := range options.BinarySection {
		writeFetchItemBinarySection(listEnc.Item(), bs)
	}
	for _, bss := range options.BinarySectionSize {
		writeFetchItemBinarySectionSize(listEnc.Item(), bss)
	}

	listEnc.End()
}

func writeFetchItemBodySection(enc *imapwire.Encoder, item *imap.FetchItemBodySection) {
	enc.Atom("BODY")
	if item.Peek {
		enc.Atom(".PEEK")
	}
	enc.Special('[')
	writeSectionPart(enc, item.Part)
	if len(item.Part) > 0 && item.Specifier != imap.PartSpecifierNone {
		enc.Special('.')
	}
	if item.Specifier != imap.PartSpecifierNone {
		enc.Atom(string(item.Specifier))

		var headerList []string
		if len(item.HeaderFields) > 0 {
			headerList = item.HeaderFields
			enc.Atom(".FIELDS")
		} else if len(item.HeaderFieldsNot) > 0 {
			headerList = item.HeaderFieldsNot
			enc.Atom(".FIELDS.NOT")
		}

		if len(headerList) > 0 {
			enc.SP().List(len(headerList), func(i int) {
				enc.String(headerList[i])
			})
		}
	}
	enc.Special(']')
	writeSectionPartial(enc, item.Partial)
}

func writeFetchItemBinarySection(enc *imapwire.Encoder, item *imap.FetchItemBinarySection) {
	enc.Atom("BINARY")
	if item.Peek {
		enc.Atom(".PEEK")
	}
	enc.Special('[')
	writeSectionPart(enc, item.Part)
	enc.Special(']')
	writeSectionPartial(enc, item.Partial)
}

func writeFetchItemBinarySectionSize(enc *imapwire.Encoder, item *imap.FetchItemBinarySectionSize) {
	enc.Atom("BINARY.SIZE")
	enc.Special('[')
	writeSectionPart(enc, item.Part)
	enc.Special(']')
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

func writeSectionPartial(enc *imapwire.Encoder, partial *imap.SectionPartial) {
	if partial == nil {
		return
	}
	enc.Special('<').Number64(partial.Offset).Special('.').Number64(partial.Size).Special('>')
}

// FetchCommand is a FETCH command.
type FetchCommand struct {
	cmd

	numSet     imap.NumSet
	recvSeqSet imap.SeqSet
	recvUIDSet imap.UIDSet

	msgs chan *FetchMessageData
	prev *FetchMessageData
}

func (cmd *FetchCommand) recvSeqNum(seqNum uint32) bool {
	set, ok := cmd.numSet.(imap.SeqSet)
	if !ok || !set.Contains(seqNum) {
		return false
	}

	if cmd.recvSeqSet.Contains(seqNum) {
		return false
	}

	cmd.recvSeqSet.AddNum(seqNum)
	return true
}

func (cmd *FetchCommand) recvUID(uid imap.UID) bool {
	set, ok := cmd.numSet.(imap.UIDSet)
	if !ok || !set.Contains(uid) {
		return false
	}

	if cmd.recvUIDSet.Contains(uid) {
		return false
	}

	cmd.recvUIDSet.AddNum(uid)
	return true
}

// Next advances to the next message.
//
// On success, the message is returned. On error or if there are no more
// messages, nil is returned. To check the error value, use Close.
func (cmd *FetchCommand) Next() *FetchMessageData {
	if cmd.prev != nil {
		cmd.prev.discard()
	}
	cmd.prev = <-cmd.msgs
	return cmd.prev
}

// Close releases the command.
//
// Calling Close unblocks the IMAP client decoder and lets it read the next
// responses. Next will always return nil after Close.
func (cmd *FetchCommand) Close() error {
	for cmd.Next() != nil {
		// ignore
	}
	return cmd.cmd.Wait()
}

// Collect accumulates message data into a list.
//
// This method will read and store message contents in memory. This is
// acceptable when the message contents have a reasonable size, but may not be
// suitable when fetching e.g. attachments.
//
// This is equivalent to calling Next repeatedly and then Close.
func (cmd *FetchCommand) Collect() ([]*FetchMessageBuffer, error) {
	defer cmd.Close()

	var l []*FetchMessageBuffer
	for {
		msg := cmd.Next()
		if msg == nil {
			break
		}

		buf, err := msg.Collect()
		if err != nil {
			return l, err
		}

		l = append(l, buf)
	}
	return l, cmd.Close()
}

// FetchMessageData contains a message's FETCH data.
type FetchMessageData struct {
	SeqNum uint32

	items chan FetchItemData
	prev  FetchItemData
}

// Next advances to the next data item for this message.
//
// If there is one or more data items left, the next item is returned.
// Otherwise nil is returned.
func (data *FetchMessageData) Next() FetchItemData {
	if d, ok := data.prev.(discarder); ok {
		d.discard()
	}

	item := <-data.items
	data.prev = item
	return item
}

func (data *FetchMessageData) discard() {
	for {
		if item := data.Next(); item == nil {
			break
		}
	}
}

// Collect accumulates message data into a struct.
//
// This method will read and store message contents in memory. This is
// acceptable when the message contents have a reasonable size, but may not be
// suitable when fetching e.g. attachments.
func (data *FetchMessageData) Collect() (*FetchMessageBuffer, error) {
	defer data.discard()

	buf := &FetchMessageBuffer{SeqNum: data.SeqNum}
	for {
		item := data.Next()
		if item == nil {
			break
		}
		if err := buf.populateItemData(item); err != nil {
			return buf, err
		}
	}
	return buf, nil
}

// FetchItemData contains a message's FETCH item data.
type FetchItemData interface {
	fetchItemData()
}

var (
	_ FetchItemData = FetchItemDataBodySection{}
	_ FetchItemData = FetchItemDataBinarySection{}
	_ FetchItemData = FetchItemDataFlags{}
	_ FetchItemData = FetchItemDataEnvelope{}
	_ FetchItemData = FetchItemDataInternalDate{}
	_ FetchItemData = FetchItemDataRFC822Size{}
	_ FetchItemData = FetchItemDataUID{}
	_ FetchItemData = FetchItemDataBodyStructure{}
)

type discarder interface {
	discard()
}

var (
	_ discarder = FetchItemDataBodySection{}
	_ discarder = FetchItemDataBinarySection{}
)

// FetchItemDataBodySection holds data returned by FETCH BODY[].
type FetchItemDataBodySection struct {
	Section *imap.FetchItemBodySection
	Literal imap.LiteralReader
}

func (FetchItemDataBodySection) fetchItemData() {}

func (item FetchItemDataBodySection) discard() {
	io.Copy(io.Discard, item.Literal)
}

// FetchItemDataBinarySection holds data returned by FETCH BINARY[].
type FetchItemDataBinarySection struct {
	Section *imap.FetchItemBinarySection
	Literal imap.LiteralReader
}

func (FetchItemDataBinarySection) fetchItemData() {}

func (item FetchItemDataBinarySection) discard() {
	io.Copy(io.Discard, item.Literal)
}

// FetchItemDataFlags holds data returned by FETCH FLAGS.
type FetchItemDataFlags struct {
	Flags []imap.Flag
}

func (FetchItemDataFlags) fetchItemData() {}

// FetchItemDataEnvelope holds data returned by FETCH ENVELOPE.
type FetchItemDataEnvelope struct {
	Envelope *imap.Envelope
}

func (FetchItemDataEnvelope) fetchItemData() {}

// FetchItemDataInternalDate holds data returned by FETCH INTERNALDATE.
type FetchItemDataInternalDate struct {
	Time time.Time
}

func (FetchItemDataInternalDate) fetchItemData() {}

// FetchItemDataRFC822Size holds data returned by FETCH RFC822.SIZE.
type FetchItemDataRFC822Size struct {
	Size int64
}

func (FetchItemDataRFC822Size) fetchItemData() {}

// FetchItemDataUID holds data returned by FETCH UID.
type FetchItemDataUID struct {
	UID imap.UID
}

func (FetchItemDataUID) fetchItemData() {}

// FetchItemDataBodyStructure holds data returned by FETCH BODYSTRUCTURE or
// FETCH BODY.
type FetchItemDataBodyStructure struct {
	BodyStructure imap.BodyStructure
	IsExtended    bool // True if BODYSTRUCTURE, false if BODY
}

func (FetchItemDataBodyStructure) fetchItemData() {}

// FetchItemDataBinarySectionSize holds data returned by FETCH BINARY.SIZE[].
type FetchItemDataBinarySectionSize struct {
	Part []int
	Size uint32
}

func (FetchItemDataBinarySectionSize) fetchItemData() {}

// FetchItemDataModSeq holds data returned by FETCH MODSEQ.
//
// This requires the CONDSTORE extension.
type FetchItemDataModSeq struct {
	ModSeq uint64
}

func (FetchItemDataModSeq) fetchItemData() {}

// FetchMessageBuffer is a buffer for the data returned by FetchMessageData.
//
// The SeqNum field is always populated. All remaining fields are optional.
type FetchMessageBuffer struct {
	SeqNum            uint32
	Flags             []imap.Flag
	Envelope          *imap.Envelope
	InternalDate      time.Time
	RFC822Size        int64
	UID               imap.UID
	BodyStructure     imap.BodyStructure
	BodySection       map[*imap.FetchItemBodySection][]byte
	BinarySection     map[*imap.FetchItemBinarySection][]byte
	BinarySectionSize []FetchItemDataBinarySectionSize
	ModSeq            uint64 // requires CONDSTORE
}

func (buf *FetchMessageBuffer) populateItemData(item FetchItemData) error {
	switch item := item.(type) {
	case FetchItemDataBodySection:
		b, err := io.ReadAll(item.Literal)
		if err != nil {
			return err
		}
		if buf.BodySection == nil {
			buf.BodySection = make(map[*imap.FetchItemBodySection][]byte)
		}
		buf.BodySection[item.Section] = b
	case FetchItemDataBinarySection:
		b, err := io.ReadAll(item.Literal)
		if err != nil {
			return err
		}
		if buf.BinarySection == nil {
			buf.BinarySection = make(map[*imap.FetchItemBinarySection][]byte)
		}
		buf.BinarySection[item.Section] = b
	case FetchItemDataFlags:
		buf.Flags = item.Flags
	case FetchItemDataEnvelope:
		buf.Envelope = item.Envelope
	case FetchItemDataInternalDate:
		buf.InternalDate = item.Time
	case FetchItemDataRFC822Size:
		buf.RFC822Size = item.Size
	case FetchItemDataUID:
		buf.UID = item.UID
	case FetchItemDataBodyStructure:
		buf.BodyStructure = item.BodyStructure
	case FetchItemDataBinarySectionSize:
		buf.BinarySectionSize = append(buf.BinarySectionSize, item)
	case FetchItemDataModSeq:
		buf.ModSeq = item.ModSeq
	default:
		panic(fmt.Errorf("unsupported fetch item data %T", item))
	}
	return nil
}

func (c *Client) handleFetch(seqNum uint32) error {
	dec := c.dec

	items := make(chan FetchItemData, 32)
	defer close(items)

	msg := &FetchMessageData{SeqNum: seqNum, items: items}

	// We're in a tricky situation: to know whether this FETCH response needs
	// to be handled by a pending command, we may need to look at the UID in
	// the response data. But the response data comes in in a streaming
	// fashion: it can contain literals. Assume that the UID will be returned
	// before any literal.
	var uid imap.UID
	handled := false
	handleMsg := func() {
		if handled {
			return
		}

		cmd := c.findPendingCmdFunc(func(anyCmd command) bool {
			cmd, ok := anyCmd.(*FetchCommand)
			if !ok {
				return false
			}

			// Skip if we haven't requested or already handled this message
			if _, ok := cmd.numSet.(imap.UIDSet); ok {
				return uid != 0 && cmd.recvUID(uid)
			} else {
				return seqNum != 0 && cmd.recvSeqNum(seqNum)
			}
		})
		if cmd != nil {
			cmd := cmd.(*FetchCommand)
			cmd.msgs <- msg
		} else if handler := c.options.unilateralDataHandler().Fetch; handler != nil {
			go handler(msg)
		} else {
			go msg.discard()
		}

		handled = true
	}
	defer handleMsg()

	numAtts := 0
	return dec.ExpectList(func() error {
		var attName string
		if !dec.Expect(dec.Func(&attName, isMsgAttNameChar), "msg-att name") {
			return dec.Err()
		}
		attName = strings.ToUpper(attName)

		var (
			item FetchItemData
			done chan struct{}
		)
		switch attName {
		case "FLAGS":
			if !dec.ExpectSP() {
				return dec.Err()
			}

			flags, err := internal.ExpectFlagList(dec)
			if err != nil {
				return err
			}

			item = FetchItemDataFlags{Flags: flags}
		case "ENVELOPE":
			if !dec.ExpectSP() {
				return dec.Err()
			}

			envelope, err := readEnvelope(dec, &c.options)
			if err != nil {
				return fmt.Errorf("in envelope: %v", err)
			}

			item = FetchItemDataEnvelope{Envelope: envelope}
		case "INTERNALDATE":
			if !dec.ExpectSP() {
				return dec.Err()
			}

			t, err := internal.ExpectDateTime(dec)
			if err != nil {
				return err
			}

			item = FetchItemDataInternalDate{Time: t}
		case "RFC822.SIZE":
			var size int64
			if !dec.ExpectSP() || !dec.ExpectNumber64(&size) {
				return dec.Err()
			}

			item = FetchItemDataRFC822Size{Size: size}
		case "UID":
			if !dec.ExpectSP() || !dec.ExpectUID(&uid) {
				return dec.Err()
			}

			item = FetchItemDataUID{UID: uid}
		case "BODY", "BINARY":
			if dec.Special('[') {
				var section interface{}
				switch attName {
				case "BODY":
					var err error
					section, err = readSectionSpec(dec)
					if err != nil {
						return fmt.Errorf("in section-spec: %v", err)
					}
				case "BINARY":
					part, dot := readSectionPart(dec)
					if dot {
						return fmt.Errorf("in section-binary: expected number after dot")
					}
					if !dec.ExpectSpecial(']') {
						return dec.Err()
					}
					section = &imap.FetchItemBinarySection{Part: part}
				}

				if !dec.ExpectSP() {
					return dec.Err()
				}

				// Ignore literal8 marker, if any
				if attName == "BINARY" {
					dec.Special('~')
				}

				lit, _, ok := dec.ExpectNStringReader()
				if !ok {
					return dec.Err()
				}

				var fetchLit imap.LiteralReader
				if lit != nil {
					done = make(chan struct{})
					fetchLit = &fetchLiteralReader{
						LiteralReader: lit,
						ch:            done,
					}
				}

				switch section := section.(type) {
				case *imap.FetchItemBodySection:
					item = FetchItemDataBodySection{
						Section: section,
						Literal: fetchLit,
					}
				case *imap.FetchItemBinarySection:
					item = FetchItemDataBinarySection{
						Section: section,
						Literal: fetchLit,
					}
				}
				break
			}
			if !dec.Expect(attName == "BODY", "'['") {
				return dec.Err()
			}
			fallthrough
		case "BODYSTRUCTURE":
			if !dec.ExpectSP() {
				return dec.Err()
			}

			bodyStruct, err := readBody(dec, &c.options)
			if err != nil {
				return err
			}

			item = FetchItemDataBodyStructure{
				BodyStructure: bodyStruct,
				IsExtended:    attName == "BODYSTRUCTURE",
			}
		case "BINARY.SIZE":
			if !dec.ExpectSpecial('[') {
				return dec.Err()
			}
			part, dot := readSectionPart(dec)
			if dot {
				return fmt.Errorf("in section-binary: expected number after dot")
			}

			var size uint32
			if !dec.ExpectSpecial(']') || !dec.ExpectSP() || !dec.ExpectNumber(&size) {
				return dec.Err()
			}

			item = FetchItemDataBinarySectionSize{
				Part: part,
				Size: size,
			}
		case "MODSEQ":
			var modSeq uint64
			if !dec.ExpectSP() || !dec.ExpectSpecial('(') || !dec.ExpectModSeq(&modSeq) || !dec.ExpectSpecial(')') {
				return dec.Err()
			}
			item = FetchItemDataModSeq{ModSeq: modSeq}
		default:
			return fmt.Errorf("unsupported msg-att name: %q", attName)
		}

		numAtts++
		if numAtts > cap(items) || done != nil {
			// To avoid deadlocking we need to ask the message handler to
			// consume the data
			handleMsg()
		}

		if done != nil {
			c.setReadTimeout(literalReadTimeout)
		}
		items <- item
		if done != nil {
			<-done
			c.setReadTimeout(respReadTimeout)
		}
		return nil
	})
}

func isMsgAttNameChar(ch byte) bool {
	return ch != '[' && imapwire.IsAtomChar(ch)
}

func readEnvelope(dec *imapwire.Decoder, options *Options) (*imap.Envelope, error) {
	var envelope imap.Envelope

	if !dec.ExpectSpecial('(') {
		return nil, dec.Err()
	}

	var date, subject string
	if !dec.ExpectNString(&date) || !dec.ExpectSP() || !dec.ExpectNString(&subject) || !dec.ExpectSP() {
		return nil, dec.Err()
	}
	// TODO: handle error
	envelope.Date, _ = netmail.ParseDate(date)
	envelope.Subject, _ = options.decodeText(subject)

	addrLists := []struct {
		name string
		out  *[]imap.Address
	}{
		{"env-from", &envelope.From},
		{"env-sender", &envelope.Sender},
		{"env-reply-to", &envelope.ReplyTo},
		{"env-to", &envelope.To},
		{"env-cc", &envelope.Cc},
		{"env-bcc", &envelope.Bcc},
	}
	for _, addrList := range addrLists {
		l, err := readAddressList(dec, options)
		if err != nil {
			return nil, fmt.Errorf("in %v: %v", addrList.name, err)
		} else if !dec.ExpectSP() {
			return nil, dec.Err()
		}
		*addrList.out = l
	}

	var inReplyTo, messageID string
	if !dec.ExpectNString(&inReplyTo) || !dec.ExpectSP() || !dec.ExpectNString(&messageID) {
		return nil, dec.Err()
	}
	// TODO: handle errors
	envelope.InReplyTo, _ = parseMsgIDList(inReplyTo)
	envelope.MessageID, _ = parseMsgID(messageID)

	if !dec.ExpectSpecial(')') {
		return nil, dec.Err()
	}
	return &envelope, nil
}

func readAddressList(dec *imapwire.Decoder, options *Options) ([]imap.Address, error) {
	var l []imap.Address
	err := dec.ExpectNList(func() error {
		addr, err := readAddress(dec, options)
		if err != nil {
			return err
		}
		l = append(l, *addr)
		return nil
	})
	return l, err
}

func readAddress(dec *imapwire.Decoder, options *Options) (*imap.Address, error) {
	var (
		addr     imap.Address
		name     string
		obsRoute string
	)
	ok := dec.ExpectSpecial('(') &&
		dec.ExpectNString(&name) && dec.ExpectSP() &&
		dec.ExpectNString(&obsRoute) && dec.ExpectSP() &&
		dec.ExpectNString(&addr.Mailbox) && dec.ExpectSP() &&
		dec.ExpectNString(&addr.Host) && dec.ExpectSpecial(')')
	if !ok {
		return nil, fmt.Errorf("in address: %v", dec.Err())
	}
	// TODO: handle error
	addr.Name, _ = options.decodeText(name)
	return &addr, nil
}

func parseMsgID(s string) (string, error) {
	var h mail.Header
	h.Set("Message-Id", s)
	return h.MessageID()
}

func parseMsgIDList(s string) ([]string, error) {
	var h mail.Header
	h.Set("In-Reply-To", s)
	return h.MsgIDList("In-Reply-To")
}

func readBody(dec *imapwire.Decoder, options *Options) (imap.BodyStructure, error) {
	if !dec.ExpectSpecial('(') {
		return nil, dec.Err()
	}

	var (
		mediaType string
		token     string
		bs        imap.BodyStructure
		err       error
	)
	if dec.String(&mediaType) {
		token = "body-type-1part"
		bs, err = readBodyType1part(dec, mediaType, options)
	} else {
		token = "body-type-mpart"
		bs, err = readBodyTypeMpart(dec, options)
	}
	if err != nil {
		return nil, fmt.Errorf("in %v: %v", token, err)
	}

	for dec.SP() {
		if !dec.DiscardValue() {
			return nil, dec.Err()
		}
	}

	if !dec.ExpectSpecial(')') {
		return nil, dec.Err()
	}

	return bs, nil
}

func readBodyType1part(dec *imapwire.Decoder, typ string, options *Options) (*imap.BodyStructureSinglePart, error) {
	bs := imap.BodyStructureSinglePart{Type: typ}

	if !dec.ExpectSP() || !dec.ExpectString(&bs.Subtype) || !dec.ExpectSP() {
		return nil, dec.Err()
	}
	var err error
	bs.Params, err = readBodyFldParam(dec, options)
	if err != nil {
		return nil, err
	}

	var description string
	if !dec.ExpectSP() || !dec.ExpectNString(&bs.ID) || !dec.ExpectSP() || !dec.ExpectNString(&description) || !dec.ExpectSP() || !dec.ExpectNString(&bs.Encoding) || !dec.ExpectSP() || !dec.ExpectBodyFldOctets(&bs.Size) {
		return nil, dec.Err()
	}

	// Content-Transfer-Encoding should always be set, but some non-standard
	// servers leave it NIL. Default to 7BIT.
	if bs.Encoding == "" {
		bs.Encoding = "7BIT"
	}

	// TODO: handle errors
	bs.Description, _ = options.decodeText(description)

	// Some servers don't include the extra fields for message and text
	// (see https://github.com/emersion/go-imap/issues/557)
	hasSP := dec.SP()
	if !hasSP {
		return &bs, nil
	}

	if strings.EqualFold(bs.Type, "message") && (strings.EqualFold(bs.Subtype, "rfc822") || strings.EqualFold(bs.Subtype, "global")) {
		var msg imap.BodyStructureMessageRFC822

		msg.Envelope, err = readEnvelope(dec, options)
		if err != nil {
			return nil, err
		}

		if !dec.ExpectSP() {
			return nil, dec.Err()
		}

		msg.BodyStructure, err = readBody(dec, options)
		if err != nil {
			return nil, err
		}

		if !dec.ExpectSP() || !dec.ExpectNumber64(&msg.NumLines) {
			return nil, dec.Err()
		}

		bs.MessageRFC822 = &msg
		hasSP = false
	} else if strings.EqualFold(bs.Type, "text") {
		var text imap.BodyStructureText

		if !dec.ExpectNumber64(&text.NumLines) {
			return nil, dec.Err()
		}

		bs.Text = &text
		hasSP = false
	}

	if !hasSP {
		hasSP = dec.SP()
	}
	if hasSP {
		bs.Extended, err = readBodyExt1part(dec, options)
		if err != nil {
			return nil, fmt.Errorf("in body-ext-1part: %v", err)
		}
	}

	return &bs, nil
}

func readBodyExt1part(dec *imapwire.Decoder, options *Options) (*imap.BodyStructureSinglePartExt, error) {
	var ext imap.BodyStructureSinglePartExt

	var md5 string
	if !dec.ExpectNString(&md5) {
		return nil, dec.Err()
	}

	if !dec.SP() {
		return &ext, nil
	}

	var err error
	ext.Disposition, err = readBodyFldDsp(dec, options)
	if err != nil {
		return nil, fmt.Errorf("in body-fld-dsp: %v", err)
	}

	if !dec.SP() {
		return &ext, nil
	}

	ext.Language, err = readBodyFldLang(dec)
	if err != nil {
		return nil, fmt.Errorf("in body-fld-lang: %v", err)
	}

	if !dec.SP() {
		return &ext, nil
	}

	if !dec.ExpectNString(&ext.Location) {
		return nil, dec.Err()
	}

	return &ext, nil
}

func readBodyTypeMpart(dec *imapwire.Decoder, options *Options) (*imap.BodyStructureMultiPart, error) {
	var bs imap.BodyStructureMultiPart

	for {
		child, err := readBody(dec, options)
		if err != nil {
			return nil, err
		}
		bs.Children = append(bs.Children, child)

		if dec.SP() && dec.String(&bs.Subtype) {
			break
		}
	}

	if dec.SP() {
		var err error
		bs.Extended, err = readBodyExtMpart(dec, options)
		if err != nil {
			return nil, fmt.Errorf("in body-ext-mpart: %v", err)
		}
	}

	return &bs, nil
}

func readBodyExtMpart(dec *imapwire.Decoder, options *Options) (*imap.BodyStructureMultiPartExt, error) {
	var ext imap.BodyStructureMultiPartExt

	var err error
	ext.Params, err = readBodyFldParam(dec, options)
	if err != nil {
		return nil, fmt.Errorf("in body-fld-param: %v", err)
	}

	if !dec.SP() {
		return &ext, nil
	}

	ext.Disposition, err = readBodyFldDsp(dec, options)
	if err != nil {
		return nil, fmt.Errorf("in body-fld-dsp: %v", err)
	}

	if !dec.SP() {
		return &ext, nil
	}

	ext.Language, err = readBodyFldLang(dec)
	if err != nil {
		return nil, fmt.Errorf("in body-fld-lang: %v", err)
	}

	if !dec.SP() {
		return &ext, nil
	}

	if !dec.ExpectNString(&ext.Location) {
		return nil, dec.Err()
	}

	return &ext, nil
}

func readBodyFldDsp(dec *imapwire.Decoder, options *Options) (*imap.BodyStructureDisposition, error) {
	if !dec.Special('(') {
		if !dec.ExpectNIL() {
			return nil, dec.Err()
		}
		return nil, nil
	}

	var disp imap.BodyStructureDisposition
	if !dec.ExpectString(&disp.Value) || !dec.ExpectSP() {
		return nil, dec.Err()
	}

	var err error
	disp.Params, err = readBodyFldParam(dec, options)
	if err != nil {
		return nil, err
	}
	if !dec.ExpectSpecial(')') {
		return nil, dec.Err()
	}
	return &disp, nil
}

func readBodyFldParam(dec *imapwire.Decoder, options *Options) (map[string]string, error) {
	var (
		params map[string]string
		k      string
	)
	err := dec.ExpectNList(func() error {
		var s string
		if !dec.ExpectString(&s) {
			return dec.Err()
		}

		if k == "" {
			k = s
		} else {
			if params == nil {
				params = make(map[string]string)
			}
			decoded, _ := options.decodeText(s)
			// TODO: handle error

			params[strings.ToLower(k)] = decoded
			k = ""
		}

		return nil
	})
	if err != nil {
		return nil, err
	} else if k != "" {
		return nil, fmt.Errorf("in body-fld-param: key without value")
	}
	return params, nil
}

func readBodyFldLang(dec *imapwire.Decoder) ([]string, error) {
	var l []string
	isList, err := dec.List(func() error {
		var s string
		if !dec.ExpectString(&s) {
			return dec.Err()
		}
		l = append(l, s)
		return nil
	})
	if err != nil || isList {
		return l, err
	}

	var s string
	if !dec.ExpectNString(&s) {
		return nil, dec.Err()
	}
	if s != "" {
		return []string{s}, nil
	} else {
		return nil, nil
	}
}

func readSectionSpec(dec *imapwire.Decoder) (*imap.FetchItemBodySection, error) {
	var section imap.FetchItemBodySection

	var dot bool
	section.Part, dot = readSectionPart(dec)
	if dot || len(section.Part) == 0 {
		var specifier string
		if dot {
			if !dec.ExpectAtom(&specifier) {
				return nil, dec.Err()
			}
		} else {
			dec.Atom(&specifier)
		}
		specifier = strings.ToUpper(specifier)
		section.Specifier = imap.PartSpecifier(specifier)

		if specifier == "HEADER.FIELDS" || specifier == "HEADER.FIELDS.NOT" {
			if !dec.ExpectSP() {
				return nil, dec.Err()
			}
			var err error
			headerList, err := readHeaderList(dec)
			if err != nil {
				return nil, err
			}
			section.Specifier = imap.PartSpecifierHeader
			if specifier == "HEADER.FIELDS" {
				section.HeaderFields = headerList
			} else {
				section.HeaderFieldsNot = headerList
			}
		}
	}

	if !dec.ExpectSpecial(']') {
		return nil, dec.Err()
	}

	offset, err := readPartialOffset(dec)
	if err != nil {
		return nil, err
	}
	if offset != nil {
		section.Partial = &imap.SectionPartial{Offset: int64(*offset)}
	}

	return &section, nil
}

func readPartialOffset(dec *imapwire.Decoder) (*uint32, error) {
	if !dec.Special('<') {
		return nil, nil
	}
	var offset uint32
	if !dec.ExpectNumber(&offset) || !dec.ExpectSpecial('>') {
		return nil, dec.Err()
	}
	return &offset, nil
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

type fetchLiteralReader struct {
	*imapwire.LiteralReader
	ch chan<- struct{}
}

func (lit *fetchLiteralReader) Read(b []byte) (int, error) {
	n, err := lit.LiteralReader.Read(b)
	if err == io.EOF && lit.ch != nil {
		close(lit.ch)
		lit.ch = nil
	}
	return n, err
}
