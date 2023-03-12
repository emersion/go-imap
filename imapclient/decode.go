package imapclient

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const dateTimeLayout = "_2-Jan-2006 15:04:05 -0700"

func readCapabilities(dec *imapwire.Decoder) (map[string]struct{}, error) {
	caps := make(map[string]struct{})
	for dec.SP() {
		var name string
		if !dec.ExpectAtom(&name) {
			return caps, fmt.Errorf("in capability-data: %v", dec.Err())
		}
		caps[name] = struct{}{}
	}
	return caps, nil
}

func readFlagList(dec *imapwire.Decoder) ([]imap.Flag, error) {
	var flags []imap.Flag
	err := dec.ExpectList(func() error {
		flag, err := readFlag(dec)
		if err != nil {
			return err
		}
		flags = append(flags, imap.Flag(flag))
		return nil
	})
	return flags, err
}

func readFlag(dec *imapwire.Decoder) (string, error) {
	isSystem := dec.Special('\\')
	var name string
	if !dec.ExpectAtom(&name) {
		return "", fmt.Errorf("in flag: %v", dec.Err())
	}
	if isSystem {
		name = "\\" + name
	}
	return name, nil
}

func readMsgAtt(dec *imapwire.Decoder, seqNum uint32, cmd *FetchCommand, options *Options) error {
	items := make(chan FetchItemData, 32)
	defer close(items)

	msg := &FetchMessageData{SeqNum: seqNum, items: items}
	if cmd != nil {
		cmd.msgs <- msg
	} else {
		defer msg.discard()
	}

	return dec.ExpectList(func() error {
		var attName string
		if !dec.ExpectAtom(&attName) {
			return dec.Err()
		}

		// TODO: BINARY section, BINARY.SIZE section
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

			envelope, err := readEnvelope(dec, options)
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
			if !dec.ExpectSP() {
				return dec.Err()
			}

			size, ok := dec.ExpectNumber64()
			if !ok {
				return dec.Err()
			}

			item = FetchItemDataRFC822Size{Size: size}
		case FetchItemUID:
			if !dec.ExpectSP() {
				return dec.Err()
			}

			uid, ok := dec.ExpectNumber()
			if !ok {
				return dec.Err()
			}

			item = FetchItemDataUID{UID: uid}
		case FetchItemBodyStructure, FetchItemBody:
			if !dec.ExpectSP() {
				return dec.Err()
			}

			bodyStruct, err := readBody(dec, options)
			if err != nil {
				return err
			}

			item = FetchItemDataBodyStructure{
				BodyStructure: bodyStruct,
				IsExtended:    attName == FetchItemBodyStructure,
			}
		case "BODY[":
			// TODO: section ["<" number ">"]
			if !dec.ExpectSpecial(']') || !dec.ExpectSP() {
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

			item = FetchItemDataSection{
				Section: nil, // TODO
				Literal: fetchLit,
			}
		case "BINARY.SIZE[":
			part, dot := readSectionPart(dec)
			if dot {
				return fmt.Errorf("expected number after dot")
			}

			if !dec.ExpectSpecial(']') || !dec.ExpectSP() {
				return dec.Err()
			}

			size, ok := dec.ExpectNumber()
			if !ok {
				return dec.Err()
			}

			item = FetchItemDataBinarySectionSize{
				Part: part,
				Size: size,
			}
		default:
			return fmt.Errorf("unsupported msg-att name: %q", attName)
		}

		items <- item
		if done != nil {
			<-done
		}
		return nil
	})
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
	if !dec.ExpectSP() || !dec.ExpectNString(&bs.ID) || !dec.ExpectSP() || !dec.ExpectNString(&description) || !dec.ExpectSP() || !dec.ExpectString(&bs.Encoding) || !dec.ExpectSP() {
		return nil, dec.Err()
	}
	// TODO: handle errors
	bs.Description, _ = options.decodeText(description)

	var ok bool
	bs.Size, ok = dec.ExpectNumber()
	if !ok {
		return nil, dec.Err()
	}

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

		if !dec.ExpectSP() {
			return nil, dec.Err()
		}

		msg.NumLines, ok = dec.ExpectNumber64()
		if !ok {
			return nil, dec.Err()
		}

		bs.MessageRFC822 = &msg
	} else if strings.EqualFold(bs.Type, "text") {
		var text BodyStructureText

		if !dec.ExpectSP() {
			return nil, dec.Err()
		}

		text.NumLines, ok = dec.ExpectNumber64()
		if !ok {
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

func readSectionPart(dec *imapwire.Decoder) (part []int, dot bool) {
	for {
		if len(part) > 0 && !dec.Special('.') {
			return part, false
		}

		num, ok := dec.Number()
		if !ok {
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

func readStatus(dec *imapwire.Decoder, cmd *StatusCommand) error {
	var data *StatusData
	if cmd != nil {
		data = &cmd.data
	} else {
		data = &StatusData{}
	}

	var err error
	if data.Mailbox, err = dec.ExpectMailbox(); err != nil {
		return err
	}

	if !dec.ExpectSP() {
		return dec.Err()
	}

	return dec.ExpectList(func() error {
		if err := readStatusAttVal(dec, data); err != nil {
			return fmt.Errorf("in status-att-val: %v", dec.Err())
		}
		return nil
	})
}

func readStatusAttVal(dec *imapwire.Decoder, data *StatusData) error {
	var name string
	if !dec.ExpectAtom(&name) || !dec.ExpectSP() {
		return dec.Err()
	}

	var ok bool
	switch StatusItem(name) {
	case StatusItemNumMessages:
		var num uint32
		num, ok = dec.ExpectNumber()
		data.NumMessages = &num
	case StatusItemUIDNext:
		data.UIDNext, ok = dec.ExpectNumber()
	case StatusItemUIDValidity:
		data.UIDValidity, ok = dec.ExpectNumber()
	case StatusItemNumUnseen:
		var num uint32
		num, ok = dec.ExpectNumber()
		data.NumUnseen = &num
	case StatusItemNumDeleted:
		var num uint32
		num, ok = dec.ExpectNumber()
		data.NumDeleted = &num
	case StatusItemSize:
		var size int64
		size, ok = dec.ExpectNumber64()
		data.Size = &size
	default:
		// TODO: skip tagged-ext
		return fmt.Errorf("unsupported status-att-val %q", name)
	}
	if !ok {
		return dec.Err()
	}
	return nil
}

func readList(dec *imapwire.Decoder) (*ListData, error) {
	var data ListData

	err := dec.ExpectList(func() error {
		attr, err := readFlag(dec)
		if err != nil {
			return err
		}
		data.Attrs = append(data.Attrs, imap.MailboxAttr(attr))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("in mbx-list-flags")
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	var delimStr string
	if dec.Quoted(&delimStr) {
		delim, size := utf8.DecodeRuneInString(delimStr)
		if delim == utf8.RuneError || size != len(delimStr) {
			return nil, fmt.Errorf("mailbox delimiter must be a single rune")
		}
		data.Delim = delim
	} else if !dec.ExpectNIL() {
		return nil, dec.Err()
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	if data.Mailbox, err = dec.ExpectMailbox(); err != nil {
		return nil, err
	}

	// TODO: [SP mbox-list-extended]

	return &data, nil
}
