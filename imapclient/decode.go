package imapclient

import (
	"fmt"
	"io"
	"time"
	"unicode/utf8"

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

func readFlagList(dec *imapwire.Decoder) ([]string, error) {
	var flags []string
	err := dec.ExpectList(func() error {
		flag, err := readFlag(dec)
		if err != nil {
			return err
		}
		flags = append(flags, flag)
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

func readMsgAtt(dec *imapwire.Decoder, seqNum uint32, cmd *FetchCommand) error {
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

		// TODO: ENVELOPE, INTERNALDATE, RFC822.SIZE, BODY, BODYSTRUCTURE,
		// BINARY section, BINARY.SIZE section, UID
		var (
			item FetchItemData
			done chan struct{}
		)
		switch FetchItem(attName) {
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

			envelope, err := readEnvelope(dec)
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

			item = FetchItemDataContents{Literal: fetchLit}
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

func readEnvelope(dec *imapwire.Decoder) (*Envelope, error) {
	var envelope Envelope

	if !dec.ExpectSpecial('(') {
		return nil, dec.Err()
	}

	if !dec.ExpectNString(&envelope.Date) || !dec.ExpectSP() || !dec.ExpectNString(&envelope.Subject) || !dec.ExpectSP() {
		return nil, dec.Err()
	}

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
		l, err := readAddressList(dec)
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

func readAddressList(dec *imapwire.Decoder) ([]Address, error) {
	var l []Address
	err := dec.ExpectNList(func() error {
		addr, err := readAddress(dec)
		if err != nil {
			return err
		}
		l = append(l, *addr)
		return nil
	})
	return l, err
}

func readAddress(dec *imapwire.Decoder) (*Address, error) {
	var (
		addr     Address
		obsRoute string
	)
	ok := dec.ExpectSpecial('(') &&
		dec.ExpectNString(&addr.Name) && dec.ExpectSP() &&
		dec.ExpectNString(&obsRoute) && dec.ExpectSP() &&
		dec.ExpectNString(&addr.Mailbox) && dec.ExpectSP() &&
		dec.ExpectNString(&addr.Host) && dec.ExpectSpecial(')')
	if !ok {
		return nil, fmt.Errorf("in address: %v", dec.Err())
	}
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

	if !dec.ExpectAString(&data.Mailbox) || !dec.ExpectSP() {
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
		data.Attrs = append(data.Attrs, MailboxAttr(attr))
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

	if !dec.ExpectSP() || !dec.ExpectAString(&data.Mailbox) {
		return nil, dec.Err()
	}

	// TODO: [SP mbox-list-extended]

	return &data, nil
}
