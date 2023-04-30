package imapmemserver

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	gomessage "github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"
)

type message struct {
	// immutable
	uid uint32
	buf []byte
	t   time.Time

	// mutable, protected by Mailbox.mutex
	flags map[imap.Flag]struct{}
}

func (msg *message) fetch(w *imapserver.FetchResponseWriter, items []imap.FetchItem) error {
	w.WriteUID(msg.uid)

	for _, item := range items {
		if err := msg.fetchItem(w, item); err != nil {
			return err
		}
	}

	return w.Close()
}

func (msg *message) fetchItem(w *imapserver.FetchResponseWriter, item imap.FetchItem) error {
	switch item := item.(type) {
	case *imap.FetchItemBodySection:
		buf := msg.bodySection(item)
		wc := w.WriteBodySection(item, int64(len(buf)))
		_, writeErr := wc.Write(buf)
		closeErr := wc.Close()
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	case *imap.FetchItemBinarySection:
		panic("TODO")
	case *imap.FetchItemBinarySectionSize:
		panic("TODO")
	}

	switch item {
	case imap.FetchItemUID:
		// always included
	case imap.FetchItemFlags:
		w.WriteFlags(msg.flagList())
	case imap.FetchItemInternalDate:
		w.WriteInternalDate(msg.t)
	case imap.FetchItemRFC822Size:
		w.WriteRFC822Size(int64(len(msg.buf)))
	case imap.FetchItemEnvelope:
		w.WriteEnvelope(msg.envelope())
	case imap.FetchItemBodyStructure, imap.FetchItemBody:
		w.WriteBodyStructure(msg.bodyStructure(item == imap.FetchItemBodyStructure))
	default:
		panic(fmt.Errorf("unknown FETCH item: %#v", item))
	}
	return nil
}

func (msg *message) envelope() *imap.Envelope {
	br := bufio.NewReader(bytes.NewReader(msg.buf))
	header, err := textproto.ReadHeader(br)
	if err != nil {
		return nil
	}
	return getEnvelope(header)
}

func (msg *message) bodyStructure(extended bool) imap.BodyStructure {
	br := bufio.NewReader(bytes.NewReader(msg.buf))
	header, _ := textproto.ReadHeader(br)
	return getBodyStructure(header, br, extended)
}

func openMessagePart(header textproto.Header, body io.Reader, parentMediaType string) (textproto.Header, io.Reader) {
	msgHeader := gomessage.Header{header}
	mediaType, _, _ := msgHeader.ContentType()
	if !msgHeader.Has("Content-Type") && parentMediaType == "multipart/digest" {
		mediaType = "message/rfc822"
	}
	if mediaType == "message/rfc822" || mediaType == "message/global" {
		br := bufio.NewReader(body)
		header, _ = textproto.ReadHeader(br)
		return header, br
	}
	return header, body
}

func (msg *message) bodySection(item *imap.FetchItemBodySection) []byte {
	var (
		header textproto.Header
		body   io.Reader
	)

	br := bufio.NewReader(bytes.NewReader(msg.buf))
	header, err := textproto.ReadHeader(br)
	if err != nil {
		return nil
	}
	body = br

	// First part of non-multipart message refers to the message itself
	msgHeader := gomessage.Header{header}
	mediaType, _, _ := msgHeader.ContentType()
	partPath := item.Part
	if !strings.HasPrefix(mediaType, "multipart/") && len(partPath) > 0 && partPath[0] == 1 {
		partPath = partPath[1:]
	}

	// Find the requested part using the provided path
	var parentMediaType string
	for i := 0; i < len(partPath); i++ {
		partNum := partPath[i]

		header, body = openMessagePart(header, body, parentMediaType)

		msgHeader := gomessage.Header{header}
		mediaType, typeParams, _ := msgHeader.ContentType()
		if !strings.HasPrefix(mediaType, "multipart/") {
			if partNum != 1 {
				return nil
			}
			continue
		}

		mr := textproto.NewMultipartReader(body, typeParams["boundary"])
		found := false
		for j := 1; j <= partNum; j++ {
			p, err := mr.NextPart()
			if err != nil {
				return nil
			}

			if j == partNum {
				parentMediaType = mediaType
				header = p.Header
				body = p
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	if len(item.Part) > 0 {
		switch item.Specifier {
		case imap.PartSpecifierHeader, imap.PartSpecifierText:
			header, body = openMessagePart(header, body, parentMediaType)
		}
	}

	// Filter header fields
	if len(item.HeaderFields) > 0 {
		keep := make(map[string]struct{})
		for _, k := range item.HeaderFields {
			keep[strings.ToLower(k)] = struct{}{}
		}
		for field := header.Fields(); field.Next(); {
			if _, ok := keep[strings.ToLower(field.Key())]; !ok {
				field.Del()
			}
		}
	}
	for _, k := range item.HeaderFieldsNot {
		header.Del(k)
	}

	// Write the requested data to a buffer
	var buf bytes.Buffer

	writeHeader := true
	switch item.Specifier {
	case imap.PartSpecifierNone:
		writeHeader = len(item.Part) == 0
	case imap.PartSpecifierText:
		writeHeader = false
	}
	if writeHeader {
		if err := textproto.WriteHeader(&buf, header); err != nil {
			return nil
		}
	}

	switch item.Specifier {
	case imap.PartSpecifierNone, imap.PartSpecifierText:
		if _, err := io.Copy(&buf, body); err != nil {
			return nil
		}
	}

	// Extract partial if any
	b := buf.Bytes()
	if partial := item.Partial; partial != nil {
		end := partial.Offset + partial.Size
		if partial.Offset > int64(len(b)) {
			return nil
		}
		if end > int64(len(b)) {
			end = int64(len(b))
		}
		b = b[partial.Offset:end]
	}
	return b
}

func (msg *message) flagList() []imap.Flag {
	var flags []imap.Flag
	for flag := range msg.flags {
		flags = append(flags, flag)
	}
	return flags
}

func (msg *message) store(store *imap.StoreFlags) {
	switch store.Op {
	case imap.StoreFlagsSet:
		msg.flags = make(map[imap.Flag]struct{})
		fallthrough
	case imap.StoreFlagsAdd:
		for _, flag := range store.Flags {
			msg.flags[canonicalFlag(flag)] = struct{}{}
		}
	case imap.StoreFlagsDel:
		for _, flag := range store.Flags {
			delete(msg.flags, canonicalFlag(flag))
		}
	default:
		panic(fmt.Errorf("unknown STORE flag operation: %v", store.Op))
	}
}

func (msg *message) search(seqNum uint32, criteria *imap.SearchCriteria) bool {
	for _, seqSet := range criteria.SeqNum {
		if seqNum == 0 || !seqSet.Contains(seqNum) {
			return false
		}
	}
	for _, seqSet := range criteria.UID {
		if !seqSet.Contains(msg.uid) {
			return false
		}
	}
	if !matchDate(msg.t, criteria.Since, criteria.Before) {
		return false
	}

	for _, flag := range criteria.Flag {
		if _, ok := msg.flags[canonicalFlag(flag)]; !ok {
			return false
		}
	}
	for _, flag := range criteria.NotFlag {
		if _, ok := msg.flags[canonicalFlag(flag)]; ok {
			return false
		}
	}

	if criteria.Larger != 0 && int64(len(msg.buf)) <= criteria.Larger {
		return false
	}
	if criteria.Smaller != 0 && int64(len(msg.buf)) >= criteria.Smaller {
		return false
	}

	if !matchBytes(msg.buf, criteria.Text) {
		return false
	}

	br := bufio.NewReader(bytes.NewReader(msg.buf))
	rawHeader, _ := textproto.ReadHeader(br)
	header := mail.Header{gomessage.Header{rawHeader}}

	for _, fieldCriteria := range criteria.Header {
		if !header.Has(fieldCriteria.Key) {
			return false
		}
		if fieldCriteria.Value == "" {
			continue
		}
		found := false
		for _, v := range header.Values(fieldCriteria.Key) {
			found = strings.Contains(strings.ToLower(v), strings.ToLower(fieldCriteria.Value))
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	if !criteria.SentSince.IsZero() || !criteria.SentBefore.IsZero() {
		t, err := header.Date()
		if err != nil {
			return false
		} else if !matchDate(t, criteria.SentSince, criteria.SentBefore) {
			return false
		}
	}

	if len(criteria.Body) > 0 {
		body, _ := io.ReadAll(br)
		if !matchBytes(body, criteria.Body) {
			return false
		}
	}

	for _, not := range criteria.Not {
		if msg.search(seqNum, &not) {
			return false
		}
	}
	for _, or := range criteria.Or {
		if !msg.search(seqNum, &or[0]) && !msg.search(seqNum, &or[1]) {
			return false
		}
	}

	return true
}

func matchDate(t, since, before time.Time) bool {
	// We discard time zone information by setting it to UTC.
	// RFC 3501 explicitly requires zone unaware date comparison.
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

	if !since.IsZero() && t.Before(since) {
		return false
	}
	if !before.IsZero() && !t.Before(before) {
		return false
	}
	return true
}

func matchBytes(buf []byte, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	buf = bytes.ToLower(buf)
	for _, s := range patterns {
		if !bytes.Contains(buf, bytes.ToLower([]byte(s))) {
			return false
		}
	}
	return true
}

func getEnvelope(h textproto.Header) *imap.Envelope {
	date, _ := netmail.ParseDate(h.Get("Date"))
	return &imap.Envelope{
		Date:      date,
		Subject:   h.Get("Subject"),
		From:      parseAddressList(h.Get("From")),
		Sender:    parseAddressList(h.Get("Sender")),
		ReplyTo:   parseAddressList(h.Get("Reply-To")),
		To:        parseAddressList(h.Get("To")),
		Cc:        parseAddressList(h.Get("Cc")),
		Bcc:       parseAddressList(h.Get("Bcc")),
		InReplyTo: h.Get("In-Reply-To"),
		MessageID: h.Get("message-Id"),
	}
}

func parseAddressList(s string) []imap.Address {
	if s == "" {
		return nil
	}

	// TODO: leave the quoted words unchanged
	// TODO: handle groups
	addrs, _ := mail.ParseAddressList(s)
	var l []imap.Address
	for _, addr := range addrs {
		mailbox, host, ok := strings.Cut(addr.Address, "@")
		if !ok {
			continue
		}
		l = append(l, imap.Address{
			Name:    mime.QEncoding.Encode("utf-8", addr.Name),
			Mailbox: mailbox,
			Host:    host,
		})
	}
	return l
}

func canonicalFlag(flag imap.Flag) imap.Flag {
	return imap.Flag(strings.ToLower(string(flag)))
}

func getBodyStructure(rawHeader textproto.Header, r io.Reader, extended bool) imap.BodyStructure {
	header := gomessage.Header{rawHeader}

	mediaType, typeParams, _ := header.ContentType()
	primaryType, subType, _ := strings.Cut(mediaType, "/")

	if primaryType == "multipart" {
		bs := &imap.BodyStructureMultiPart{Subtype: subType}
		mr := textproto.NewMultipartReader(r, typeParams["boundary"])
		for {
			part, _ := mr.NextPart()
			if part == nil {
				break
			}
			bs.Children = append(bs.Children, getBodyStructure(part.Header, part, extended))
		}
		if extended {
			bs.Extended = &imap.BodyStructureMultiPartExt{
				Params:      typeParams,
				Disposition: getContentDisposition(header),
				Language:    getContentLanguage(header),
				Location:    header.Get("Content-Location"),
			}
		}
		return bs
	} else {
		body, _ := io.ReadAll(r) // TODO: optimize
		bs := &imap.BodyStructureSinglePart{
			Type:        primaryType,
			Subtype:     subType,
			Params:      typeParams,
			ID:          header.Get("Content-Id"),
			Description: header.Get("Content-Description"),
			Encoding:    header.Get("Content-Transfer-Encoding"),
			Size:        uint32(len(body)),
		}
		if mediaType == "message/rfc822" || mediaType == "message/global" {
			br := bufio.NewReader(bytes.NewReader(body))
			childHeader, _ := textproto.ReadHeader(br)
			bs.MessageRFC822 = &imap.BodyStructureMessageRFC822{
				Envelope:      getEnvelope(childHeader),
				BodyStructure: getBodyStructure(childHeader, br, extended),
				NumLines:      int64(bytes.Count(body, []byte("\n"))),
			}
		}
		if primaryType == "text" {
			bs.Text = &imap.BodyStructureText{
				NumLines: int64(bytes.Count(body, []byte("\n"))),
			}
		}
		if extended {
			bs.Extended = &imap.BodyStructureSinglePartExt{
				Disposition: getContentDisposition(header),
				Language:    getContentLanguage(header),
				Location:    header.Get("Content-Location"),
			}
		}
		return bs
	}
}

func getContentDisposition(header gomessage.Header) *imap.BodyStructureDisposition {
	disp, dispParams, _ := header.ContentDisposition()
	if disp == "" {
		return nil
	}
	return &imap.BodyStructureDisposition{
		Value:  disp,
		Params: dispParams,
	}
}

func getContentLanguage(header gomessage.Header) []string {
	v := header.Get("Content-Language")
	if v == "" {
		return nil
	}
	// TODO: handle CFWS
	l := strings.Split(v, ",")
	for i, lang := range l {
		l[i] = strings.TrimSpace(lang)
	}
	return l
}
