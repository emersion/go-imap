package imapserver

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	gomessage "github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"

	"github.com/emersion/go-imap/v2"
)

// ExtractBodySection extracts a section of a message body.
//
// It can be used by server backends to implement Session.Fetch.
func ExtractBodySection(r io.Reader, item *imap.FetchItemBodySection) []byte {
	var (
		header textproto.Header
		body   io.Reader
	)

	br := bufio.NewReader(r)
	header, err := textproto.ReadHeader(br)
	if err != nil {
		return nil
	}
	body = br

	parentMediaType, header, body := findMessagePart(header, body, item.Part)
	if body == nil {
		return nil
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

	return extractPartial(buf.Bytes(), item.Partial)
}

func findMessagePart(header textproto.Header, body io.Reader, partPath []int) (string, textproto.Header, io.Reader) {
	// First part of non-multipart message refers to the message itself
	msgHeader := gomessage.Header{header}
	mediaType, _, _ := msgHeader.ContentType()
	if !strings.HasPrefix(mediaType, "multipart/") && len(partPath) > 0 && partPath[0] == 1 {
		partPath = partPath[1:]
	}

	var parentMediaType string
	for i := 0; i < len(partPath); i++ {
		partNum := partPath[i]

		header, body = openMessagePart(header, body, parentMediaType)

		msgHeader := gomessage.Header{header}
		mediaType, typeParams, _ := msgHeader.ContentType()
		if !strings.HasPrefix(mediaType, "multipart/") {
			if partNum != 1 {
				return "", textproto.Header{}, nil
			}
			continue
		}

		mr := textproto.NewMultipartReader(body, typeParams["boundary"])
		found := false
		for j := 1; j <= partNum; j++ {
			p, err := mr.NextPart()
			if err != nil {
				return "", textproto.Header{}, nil
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
			return "", textproto.Header{}, nil
		}
	}

	return parentMediaType, header, body
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

func extractPartial(b []byte, partial *imap.SectionPartial) []byte {
	if partial == nil {
		return b
	}

	end := partial.Offset + partial.Size
	if partial.Offset > int64(len(b)) {
		return nil
	}
	if end > int64(len(b)) {
		end = int64(len(b))
	}
	return b[partial.Offset:end]
}

func ExtractBinarySection(r io.Reader, item *imap.FetchItemBinarySection) []byte {
	var (
		header textproto.Header
		body   io.Reader
	)

	br := bufio.NewReader(r)
	header, err := textproto.ReadHeader(br)
	if err != nil {
		return nil
	}
	body = br

	_, header, body = findMessagePart(header, body, item.Part)
	if body == nil {
		return nil
	}

	part, err := gomessage.New(gomessage.Header{header}, body)
	if err != nil {
		return nil
	}

	// Write the requested data to a buffer
	var buf bytes.Buffer

	if len(item.Part) == 0 {
		if err := textproto.WriteHeader(&buf, part.Header.Header); err != nil {
			return nil
		}
	}

	if _, err := io.Copy(&buf, part.Body); err != nil {
		return nil
	}

	return extractPartial(buf.Bytes(), item.Partial)
}

func ExtractBinarySectionSize(r io.Reader, item *imap.FetchItemBinarySectionSize) uint32 {
	// TODO: optimize
	b := ExtractBinarySection(r, &imap.FetchItemBinarySection{Part: item.Part})
	return uint32(len(b))
}

// ExtractEnvelope returns a message envelope from its header.
//
// It can be used by server backends to implement Session.Fetch.
func ExtractEnvelope(h textproto.Header) *imap.Envelope {
	mh := mail.Header{gomessage.Header{h}}
	date, _ := mh.Date()
	subject, _ := mh.Subject()
	inReplyTo, _ := mh.MsgIDList("In-Reply-To")
	messageID, _ := mh.MessageID()
	return &imap.Envelope{
		Date:      date,
		Subject:   subject,
		From:      parseAddressList(mh, "From"),
		Sender:    parseAddressList(mh, "Sender"),
		ReplyTo:   parseAddressList(mh, "Reply-To"),
		To:        parseAddressList(mh, "To"),
		Cc:        parseAddressList(mh, "Cc"),
		Bcc:       parseAddressList(mh, "Bcc"),
		InReplyTo: inReplyTo,
		MessageID: messageID,
	}
}

func parseAddressList(mh mail.Header, k string) []imap.Address {
	// TODO: handle groups
	addrs, _ := mh.AddressList(k)
	var l []imap.Address
	for _, addr := range addrs {
		mailbox, host, ok := strings.Cut(addr.Address, "@")
		if !ok {
			continue
		}
		l = append(l, imap.Address{
			Name:    addr.Name,
			Mailbox: mailbox,
			Host:    host,
		})
	}
	return l
}

// ExtractBodyStructure extracts the structure of a message body.
//
// It can be used by server backends to implement Session.Fetch.
func ExtractBodyStructure(r io.Reader) imap.BodyStructure {
	br := bufio.NewReader(r)
	header, _ := textproto.ReadHeader(br)
	return extractBodyStructure(header, br)
}

func extractBodyStructure(rawHeader textproto.Header, r io.Reader) imap.BodyStructure {
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
			bs.Children = append(bs.Children, extractBodyStructure(part.Header, part))
		}
		bs.Extended = &imap.BodyStructureMultiPartExt{
			Params:      typeParams,
			Disposition: getContentDisposition(header),
			Language:    getContentLanguage(header),
			Location:    header.Get("Content-Location"),
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
				Envelope:      ExtractEnvelope(childHeader),
				BodyStructure: extractBodyStructure(childHeader, br),
				NumLines:      int64(bytes.Count(body, []byte("\n"))),
			}
		}
		if primaryType == "text" {
			bs.Text = &imap.BodyStructureText{
				NumLines: int64(bytes.Count(body, []byte("\n"))),
			}
		}
		bs.Extended = &imap.BodyStructureSinglePartExt{
			Disposition: getContentDisposition(header),
			Language:    getContentLanguage(header),
			Location:    header.Get("Content-Location"),
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
