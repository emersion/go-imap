package imapclient

import (
	"fmt"
	"io"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

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
	if !dec.ExpectSpecial('(') {
		return nil, dec.Err()
	}
	if dec.Special(')') {
		return nil, nil
	}

	flag, err := readFlag(dec)
	if err != nil {
		return nil, err
	}

	flags := []string{flag}
	for dec.SP() {
		flag, err := readFlag(dec)
		if err != nil {
			return flags, err
		}
		flags = append(flags, flag)
	}

	if !dec.ExpectSpecial(')') {
		return flags, dec.Err()
	}
	return flags, nil
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

func readMsgAtt(dec *imapwire.Decoder, cmd *FetchCommand) error {
	if !dec.ExpectSpecial('(') {
		return dec.Err()
	}

	items := make(chan *FetchItemData, 32)
	defer close(items)
	msg := &FetchMessageData{items: items}

	if cmd != nil {
		cmd.msgs <- msg
	} else {
		defer msg.discard()
	}

	for {
		var attName string
		if !dec.ExpectAtom(&attName) {
			return dec.Err()
		}

		// TODO: FLAGS, ENVELOPE, INTERNALDATE, RFC822.SIZE, BODY,
		// BODYSTRUCTURE, BINARY section, BINARY.SIZE section, UID
		var (
			item *FetchItemData
			done chan struct{}
		)
		switch attName {
		case "BODY[":
			// TODO: section ["<" number ">"]
			if !dec.ExpectSpecial(']') || !dec.ExpectSP() {
				return dec.Err()
			}

			lit, _, ok := dec.ExpectNString()
			if !ok {
				return dec.Err()
			}

			done = make(chan struct{})
			item = &FetchItemData{
				Name: "BODY[]",
				Literal: &fetchLiteralReader{
					LiteralReader: lit,
					ch:            done,
				},
			}
		default:
			return fmt.Errorf("unsupported msg-att name: %q", attName)
		}

		items <- item
		if done != nil {
			<-done
		}

		if !dec.SP() {
			break
		}
	}

	if !dec.ExpectSpecial(')') {
		return dec.Err()
	}

	return nil
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

	if !dec.ExpectAString(&data.Mailbox) || !dec.ExpectSP() || !dec.ExpectSpecial('(') {
		return dec.Err()
	}

	for {
		if err := readStatusAttVal(dec, data); err != nil {
			return fmt.Errorf("in status-att-val: %v", dec.Err())
		}

		if !dec.SP() {
			break
		}
	}

	if !dec.ExpectSpecial(')') {
		return dec.Err()
	}
	return nil
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
