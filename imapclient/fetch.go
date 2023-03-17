package imapclient

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Client) fetch(uid bool, seqSet imap.SeqSet, items []FetchItem) *FetchCommand {
	// Ensure we request UID as the first data item for UID FETCH, to be safer.
	// We want to get it before any literal.
	if uid {
		itemsWithUID := []FetchItem{FetchItemUID}
		for _, item := range items {
			if item != FetchItemUID {
				itemsWithUID = append(itemsWithUID, item)
			}
		}
		items = itemsWithUID
	}

	cmd := &FetchCommand{
		uid:    uid,
		seqSet: seqSet,
		msgs:   make(chan *FetchMessageData, 128),
	}
	enc := c.beginCommand(uidCmdName("FETCH", uid), cmd)
	enc.SP().Atom(seqSet.String()).SP().List(len(items), func(i int) {
		writeFetchItem(enc.Encoder, items[i])
	})
	enc.end()
	return cmd
}

// Fetch sends a FETCH command.
//
// The caller must fully consume the FetchCommand. A simple way to do so is to
// defer a call to FetchCommand.Close.
func (c *Client) Fetch(seqSet imap.SeqSet, items []FetchItem) *FetchCommand {
	return c.fetch(false, seqSet, items)
}

// UIDFetch sends a UID FETCH command.
//
// See Fetch.
func (c *Client) UIDFetch(seqSet imap.SeqSet, items []FetchItem) *FetchCommand {
	return c.fetch(true, seqSet, items)
}

// FetchItem is a message data item which can be requested by a FETCH command.
type FetchItem interface {
	fetchItem()
}

var (
	_ FetchItem = FetchItemKeyword("")
	_ FetchItem = (*FetchItemBodySection)(nil)
	_ FetchItem = (*FetchItemBinarySection)(nil)
	_ FetchItem = (*FetchItemBinarySectionSize)(nil)
)

// FetchItemKeyword is a FETCH item described by a single keyword.
type FetchItemKeyword string

func (FetchItemKeyword) fetchItem() {}

var (
	// Macros
	FetchItemAll  FetchItem = FetchItemKeyword("ALL")
	FetchItemFast FetchItem = FetchItemKeyword("FAST")
	FetchItemFull FetchItem = FetchItemKeyword("FULL")

	FetchItemBody          FetchItem = FetchItemKeyword("BODY")
	FetchItemBodyStructure FetchItem = FetchItemKeyword("BODYSTRUCTURE")
	FetchItemEnvelope      FetchItem = FetchItemKeyword("ENVELOPE")
	FetchItemFlags         FetchItem = FetchItemKeyword("FLAGS")
	FetchItemInternalDate  FetchItem = FetchItemKeyword("INTERNALDATE")
	FetchItemRFC822Size    FetchItem = FetchItemKeyword("RFC822.SIZE")
	FetchItemUID           FetchItem = FetchItemKeyword("UID")
)

type PartSpecifier string

const (
	PartSpecifierNone   PartSpecifier = ""
	PartSpecifierHeader PartSpecifier = "HEADER"
	PartSpecifierMIME   PartSpecifier = "MIME"
	PartSpecifierText   PartSpecifier = "TEXT"
)

type SectionPartial struct {
	Offset, Size int64
}

// FetchItemBodySection is a FETCH BODY[] data item.
type FetchItemBodySection struct {
	Specifier       PartSpecifier
	Part            []int
	HeaderFields    []string
	HeaderFieldsNot []string
	Partial         *SectionPartial
	Peek            bool
}

func (*FetchItemBodySection) fetchItem() {}

// FetchItemBinarySection is a FETCH BINARY[] data item.
type FetchItemBinarySection struct {
	Part    []int
	Partial *SectionPartial
	Peek    bool
}

func (*FetchItemBinarySection) fetchItem() {}

// FetchItemBinarySectionSize is a FETCH BINARY.SIZE[] data item.
type FetchItemBinarySectionSize struct {
	Part []int
}

func (*FetchItemBinarySectionSize) fetchItem() {}

func writeFetchItem(enc *imapwire.Encoder, item FetchItem) {
	switch item := item.(type) {
	case FetchItemKeyword:
		enc.Atom(string(item))
	case *FetchItemBodySection:
		enc.Atom("BODY")
		if item.Peek {
			enc.Atom(".PEEK")
		}
		enc.Special('[')
		writeSectionPart(enc, item.Part)
		if len(item.Part) > 0 && item.Specifier != PartSpecifierNone {
			enc.Special('.')
		}
		if item.Specifier != PartSpecifierNone {
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
	case *FetchItemBinarySection:
		enc.Atom("BINARY")
		if item.Peek {
			enc.Atom(".PEEK")
		}
		enc.Special('[')
		writeSectionPart(enc, item.Part)
		enc.Special(']')
		writeSectionPartial(enc, item.Partial)
	case *FetchItemBinarySectionSize:
		enc.Atom("BINARY.SIZE")
		enc.Special('[')
		writeSectionPart(enc, item.Part)
		enc.Special(']')
	default:
		panic(fmt.Errorf("imapclient: unknown fetch item type %T", item))
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

func writeSectionPartial(enc *imapwire.Encoder, partial *SectionPartial) {
	if partial == nil {
		return
	}
	enc.Special('<').Number64(partial.Offset).Special('.').Number64(partial.Size).Special('>')
}

// FetchCommand is a FETCH command.
type FetchCommand struct {
	cmd

	uid        bool
	seqSet     imap.SeqSet
	recvSeqSet imap.SeqSet

	msgs chan *FetchMessageData
	prev *FetchMessageData
}

// Next advances to the next message.
//
// On success, the message is returned. On error or if there are no more
// messages, nil is returned. To check the error value, use Close.
func (cmd *FetchCommand) Next() *FetchMessageData {
	if cmd.prev != nil {
		cmd.prev.discard()
	}
	return <-cmd.msgs
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
	Section *FetchItemBodySection
	Literal LiteralReader
}

func (FetchItemDataBodySection) fetchItemData() {}

func (item FetchItemDataBodySection) discard() {
	io.Copy(io.Discard, item.Literal)
}

// FetchItemDataBinarySection holds data returned by FETCH BINARY[].
type FetchItemDataBinarySection struct {
	Section *FetchItemBinarySection
	Literal LiteralReader
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
	Envelope *Envelope
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
	UID uint32
}

func (FetchItemDataUID) fetchItemData() {}

// FetchItemDataBodyStructure holds data returned by FETCH BODYSTRUCTURE or
// FETCH BODY.
type FetchItemDataBodyStructure struct {
	BodyStructure BodyStructure
	IsExtended    bool // True if BODYSTRUCTURE, false if BODY
}

func (FetchItemDataBodyStructure) fetchItemData() {}

// FetchItemDataBinarySectionSize holds data returned by FETCH BINARY.SIZE[].
type FetchItemDataBinarySectionSize struct {
	Part []int
	Size uint32
}

func (FetchItemDataBinarySectionSize) fetchItemData() {}

// Envelope is the envelope structure of a message.
type Envelope struct {
	Date      string // see net/mail.ParseDate
	Subject   string
	From      []Address
	Sender    []Address
	ReplyTo   []Address
	To        []Address
	Cc        []Address
	Bcc       []Address
	InReplyTo string
	MessageID string
}

// Address represents a sender or recipient of a message.
type Address struct {
	Name    string
	Mailbox string
	Host    string
}

// Addr returns the e-mail address in the form "foo@example.org".
//
// If the address is a start or end of group, the empty string is returned.
func (addr *Address) Addr() string {
	if addr.Mailbox == "" || addr.Host == "" {
		return ""
	}
	return addr.Mailbox + "@" + addr.Host
}

// IsGroupStart returns true if this address is a start of group marker.
//
// In that case, Mailbox contains the group name phrase.
func (addr *Address) IsGroupStart() bool {
	return addr.Host == "" && addr.Mailbox != ""
}

// IsGroupEnd returns true if this address is a end of group marker.
func (addr *Address) IsGroupEnd() bool {
	return addr.Host == "" && addr.Mailbox == ""
}

// BodyStructure describes the body structure of a message.
//
// A BodyStructure value is either a *BodyStructureSinglePart or a
// *BodyStructureMultiPart.
type BodyStructure interface {
	// MediaType returns the MIME type of this body structure, e.g. "text/plain".
	MediaType() string
	// Walk walks the body structure tree, calling f for each part in the tree,
	// including bs itself. The parts are visited in DFS pre-order.
	Walk(f BodyStructureWalkFunc)
	// Disposition returns the body structure disposition, if available.
	Disposition() *BodyStructureDisposition

	bodyStructure()
}

var (
	_ BodyStructure = (*BodyStructureSinglePart)(nil)
	_ BodyStructure = (*BodyStructureMultiPart)(nil)
)

// BodyStructureSinglePart is a body structure with a single part.
type BodyStructureSinglePart struct {
	Type, Subtype string
	Params        map[string]string
	ID            string
	Description   string
	Encoding      string
	Size          uint32

	MessageRFC822 *BodyStructureMessageRFC822 // only for "message/rfc822"
	Text          *BodyStructureText          // only for "text/*"
	Extended      *BodyStructureSinglePartExt
}

func (bs *BodyStructureSinglePart) MediaType() string {
	return strings.ToLower(bs.Type) + "/" + strings.ToLower(bs.Subtype)
}

func (bs *BodyStructureSinglePart) Walk(f BodyStructureWalkFunc) {
	f([]int{1}, bs)
}

func (bs *BodyStructureSinglePart) Disposition() *BodyStructureDisposition {
	if bs.Extended == nil {
		return nil
	}
	return bs.Extended.Disposition
}

// Filename decodes the body structure's filename, if any.
func (bs *BodyStructureSinglePart) Filename() string {
	var filename string
	if bs.Extended != nil && bs.Extended.Disposition != nil {
		filename = bs.Extended.Disposition.Params["filename"]
	}
	if filename == "" {
		// Note: using "name" in Content-Type is discouraged
		filename = bs.Params["name"]
	}
	return filename
}

func (*BodyStructureSinglePart) bodyStructure() {}

type BodyStructureMessageRFC822 struct {
	Envelope      *Envelope
	BodyStructure BodyStructure
	NumLines      int64
}

type BodyStructureText struct {
	NumLines int64
}

type BodyStructureSinglePartExt struct {
	MD5         string
	Disposition *BodyStructureDisposition
	Language    []string
	Location    string
}

// BodyStructureMultiPart is a body structure with multiple parts.
type BodyStructureMultiPart struct {
	Children []BodyStructure
	Subtype  string

	Extended *BodyStructureMultiPartExt
}

func (bs *BodyStructureMultiPart) MediaType() string {
	return "multipart/" + strings.ToLower(bs.Subtype)
}

func (bs *BodyStructureMultiPart) Walk(f BodyStructureWalkFunc) {
	bs.walk(f, nil)
}

func (bs *BodyStructureMultiPart) walk(f BodyStructureWalkFunc, path []int) {
	if !f(path, bs) {
		return
	}

	pathBuf := make([]int, len(path))
	copy(pathBuf, path)
	for i, part := range bs.Children {
		num := i + 1
		partPath := append(pathBuf, num)

		switch part := part.(type) {
		case *BodyStructureSinglePart:
			f(partPath, part)
		case *BodyStructureMultiPart:
			part.walk(f, partPath)
		default:
			panic(fmt.Errorf("unsupported body structure type %T", part))
		}
	}
}

func (bs *BodyStructureMultiPart) Disposition() *BodyStructureDisposition {
	if bs.Extended == nil {
		return nil
	}
	return bs.Extended.Disposition
}

func (*BodyStructureMultiPart) bodyStructure() {}

type BodyStructureMultiPartExt struct {
	Params      map[string]string
	Disposition *BodyStructureDisposition
	Language    []string
	Location    string
}

type BodyStructureDisposition struct {
	Value  string
	Params map[string]string
}

// BodyStructureWalkFunc is a function called for each body structure visited
// by BodyStructure.Walk.
//
// The path argument contains the IMAP part path.
//
// The function should return true to visit all of the part's children or false
// to skip them.
type BodyStructureWalkFunc func(path []int, part BodyStructure) (walkChildren bool)

// LiteralReader is a reader for IMAP literals.
type LiteralReader interface {
	io.Reader
	Size() int64
}

// FetchMessageBuffer is a buffer for the data returned by FetchMessageData.
//
// The SeqNum field is always populated. All remaining fields are optional.
type FetchMessageBuffer struct {
	SeqNum            uint32
	Flags             []imap.Flag
	Envelope          *Envelope
	InternalDate      time.Time
	RFC822Size        int64
	UID               uint32
	BodyStructure     BodyStructure
	BodySection       map[*FetchItemBodySection][]byte
	BinarySection     map[*FetchItemBinarySection][]byte
	BinarySectionSize []FetchItemDataBinarySectionSize
}

func (buf *FetchMessageBuffer) populateItemData(item FetchItemData) error {
	switch item := item.(type) {
	case FetchItemDataBodySection:
		b, err := io.ReadAll(item.Literal)
		if err != nil {
			return err
		}
		if buf.BodySection == nil {
			buf.BodySection = make(map[*FetchItemBodySection][]byte)
		}
		buf.BodySection[item.Section] = b
	case FetchItemDataBinarySection:
		b, err := io.ReadAll(item.Literal)
		if err != nil {
			return err
		}
		if buf.BinarySection == nil {
			buf.BinarySection = make(map[*FetchItemBinarySection][]byte)
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
	default:
		panic(fmt.Errorf("unsupported fetch item data %T", item))
	}
	return nil
}

func readMsgAtt(c *Client, seqNum uint32) error {
	dec := c.dec

	items := make(chan FetchItemData, 32)
	defer close(items)

	msg := &FetchMessageData{SeqNum: seqNum, items: items}

	// We're in a tricky situation: to know whether this FETCH response needs
	// to be handled by a pending command, we may need to look at the UID in
	// the response data. But the response data comes in in a streaming
	// fashion: it can contain literals. Assume that the UID will be returned
	// before any literal.
	var uid uint32
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
			var num uint32
			if cmd.uid {
				num = uid
			} else {
				num = seqNum
			}
			if num == 0 || !cmd.seqSet.Contains(num) || cmd.recvSeqSet.Contains(num) {
				return false
			}
			cmd.recvSeqSet.AddNum(num)

			return true
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

		var (
			item FetchItemData
			done chan struct{}
		)
		switch attName := FetchItemKeyword(attName); attName {
		case FetchItemFlags:
			if !dec.ExpectSP() {
				return dec.Err()
			}

			flags, err := readFlagList(dec)
			if err != nil {
				return err
			}

			item = FetchItemDataFlags{Flags: flags}
		case FetchItemEnvelope:
			if !dec.ExpectSP() {
				return dec.Err()
			}

			envelope, err := readEnvelope(dec, &c.options)
			if err != nil {
				return fmt.Errorf("in envelope: %v", err)
			}

			item = FetchItemDataEnvelope{Envelope: envelope}
		case FetchItemInternalDate:
			if !dec.ExpectSP() {
				return dec.Err()
			}

			t, err := readDateTime(dec)
			if err != nil {
				return err
			}

			item = FetchItemDataInternalDate{Time: t}
		case FetchItemRFC822Size:
			var size int64
			if !dec.ExpectSP() || !dec.ExpectNumber64(&size) {
				return dec.Err()
			}

			item = FetchItemDataRFC822Size{Size: size}
		case FetchItemUID:
			if !dec.ExpectSP() || !dec.ExpectNumber(&uid) {
				return dec.Err()
			}

			item = FetchItemDataUID{UID: uid}
		case "BODY", "BINARY":
			if dec.Special('[') {
				var section FetchItem
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
					section = &FetchItemBinarySection{Part: part}
				}

				if !dec.ExpectSP() {
					return dec.Err()
				}

				lit, _, ok := dec.ExpectNStringReader()
				if !ok {
					return dec.Err()
				}

				var fetchLit LiteralReader
				if lit != nil {
					done = make(chan struct{})
					fetchLit = &fetchLiteralReader{
						LiteralReader: lit,
						ch:            done,
					}
				}

				switch section := section.(type) {
				case *FetchItemBodySection:
					item = FetchItemDataBodySection{
						Section: section,
						Literal: fetchLit,
					}
				case *FetchItemBinarySection:
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
		case FetchItemBodyStructure:
			if !dec.ExpectSP() {
				return dec.Err()
			}

			bodyStruct, err := readBody(dec, &c.options)
			if err != nil {
				return err
			}

			item = FetchItemDataBodyStructure{
				BodyStructure: bodyStruct,
				IsExtended:    attName == FetchItemBodyStructure,
			}
		case "BINARY.SIZE":
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

func readEnvelope(dec *imapwire.Decoder, options *Options) (*Envelope, error) {
	var envelope Envelope

	if !dec.ExpectSpecial('(') {
		return nil, dec.Err()
	}

	var subject string
	if !dec.ExpectNString(&envelope.Date) || !dec.ExpectSP() || !dec.ExpectNString(&subject) || !dec.ExpectSP() {
		return nil, dec.Err()
	}
	// TODO: handle error
	envelope.Subject, _ = options.decodeText(subject)

	addrLists := []struct {
		name string
		out  *[]Address
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

	if !dec.ExpectNString(&envelope.InReplyTo) || !dec.ExpectSP() || !dec.ExpectNString(&envelope.MessageID) {
		return nil, dec.Err()
	}

	if !dec.ExpectSpecial(')') {
		return nil, dec.Err()
	}
	return &envelope, nil
}

func readAddressList(dec *imapwire.Decoder, options *Options) ([]Address, error) {
	var l []Address
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

func readAddress(dec *imapwire.Decoder, options *Options) (*Address, error) {
	var (
		addr     Address
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

func readDateTime(dec *imapwire.Decoder) (time.Time, error) {
	var s string
	if !dec.Expect(dec.Quoted(&s), "date-time") {
		return time.Time{}, dec.Err()
	}
	t, err := time.Parse(dateTimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("in date-time: %v", err)
	}
	return t, err
}

func readBody(dec *imapwire.Decoder, options *Options) (BodyStructure, error) {
	if !dec.ExpectSpecial('(') {
		return nil, dec.Err()
	}

	var (
		mediaType string
		token     string
		bs        BodyStructure
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

	// TODO: skip all unread fields until ')'
	if !dec.ExpectSpecial(')') {
		return nil, dec.Err()
	}

	return bs, nil
}

func readBodyType1part(dec *imapwire.Decoder, typ string, options *Options) (*BodyStructureSinglePart, error) {
	bs := BodyStructureSinglePart{Type: typ}

	if !dec.ExpectSP() || !dec.ExpectString(&bs.Subtype) || !dec.ExpectSP() {
		return nil, dec.Err()
	}

	var err error
	bs.Params, err = readBodyFldParam(dec)
	if err != nil {
		return nil, err
	}
	if name, ok := bs.Params["name"]; ok {
		// TODO: handle error
		bs.Params["name"], _ = options.decodeText(name)
	}

	var description string
	if !dec.ExpectSP() || !dec.ExpectNString(&bs.ID) || !dec.ExpectSP() || !dec.ExpectNString(&description) || !dec.ExpectSP() || !dec.ExpectString(&bs.Encoding) || !dec.ExpectSP() || !dec.ExpectNumber(&bs.Size) {
		return nil, dec.Err()
	}
	// TODO: handle errors
	bs.Description, _ = options.decodeText(description)

	if strings.EqualFold(bs.Type, "message") && (strings.EqualFold(bs.Subtype, "rfc822") || strings.EqualFold(bs.Subtype, "global")) {
		var msg BodyStructureMessageRFC822

		if !dec.ExpectSP() {
			return nil, dec.Err()
		}

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
	} else if strings.EqualFold(bs.Type, "text") {
		var text BodyStructureText

		if !dec.ExpectSP() || !dec.ExpectNumber64(&text.NumLines) {
			return nil, dec.Err()
		}

		bs.Text = &text
	}

	if dec.SP() {
		bs.Extended, err = readBodyExt1part(dec, options)
		if err != nil {
			return nil, fmt.Errorf("in body-ext-1part: %v", err)
		}
	}

	return &bs, nil
}

func readBodyExt1part(dec *imapwire.Decoder, options *Options) (*BodyStructureSinglePartExt, error) {
	var ext BodyStructureSinglePartExt

	if !dec.ExpectNString(&ext.MD5) {
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

func readBodyTypeMpart(dec *imapwire.Decoder, options *Options) (*BodyStructureMultiPart, error) {
	var bs BodyStructureMultiPart

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

func readBodyExtMpart(dec *imapwire.Decoder, options *Options) (*BodyStructureMultiPartExt, error) {
	var ext BodyStructureMultiPartExt

	var err error
	ext.Params, err = readBodyFldParam(dec)
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

func readBodyFldDsp(dec *imapwire.Decoder, options *Options) (*BodyStructureDisposition, error) {
	if !dec.Special('(') {
		if !dec.ExpectNIL() {
			return nil, dec.Err()
		}
		return nil, nil
	}

	var disp BodyStructureDisposition
	if !dec.ExpectString(&disp.Value) || !dec.ExpectSP() {
		return nil, dec.Err()
	}

	var err error
	disp.Params, err = readBodyFldParam(dec)
	if err != nil {
		return nil, err
	}
	if filename, ok := disp.Params["filename"]; ok {
		// TODO: handle error
		disp.Params["filename"], _ = options.decodeText(filename)
	}

	if !dec.ExpectSpecial(')') {
		return nil, dec.Err()
	}
	return &disp, nil
}

func readBodyFldParam(dec *imapwire.Decoder) (map[string]string, error) {
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
			params[k] = s
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

func readSectionSpec(dec *imapwire.Decoder) (*FetchItemBodySection, error) {
	var section FetchItemBodySection

	var dot bool
	section.Part, dot = readSectionPart(dec)
	if dot {
		var specifier string
		if !dec.ExpectAtom(&specifier) {
			return nil, dec.Err()
		}
		section.Specifier = PartSpecifier(specifier)

		if specifier == "HEADER.FIELDS" || specifier == "HEADER.FIELDS.NOT" {
			if !dec.ExpectSP() {
				return nil, dec.Err()
			}
			var err error
			headerList, err := readHeaderList(dec)
			if err != nil {
				return nil, err
			}
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
		section.Partial = &SectionPartial{Offset: int64(*offset)}
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
