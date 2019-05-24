package backendutil

import (
	"bytes"
	"errors"
	"io"
	"net/textproto"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
)

var errNoSuchPart = errors.New("backendutil: no such message body part")

// FetchBodySection extracts a body section from a message.
func FetchBodySection(e *message.Entity, section *imap.BodySectionName) (imap.Literal, error) {
	// First, find the requested part using the provided path
	for i := 0; i < len(section.Path); i++ {
		n := section.Path[i]

		mr := e.MultipartReader()
		if mr == nil {
			return nil, errNoSuchPart
		}

		for j := 1; j <= n; j++ {
			p, err := mr.NextPart()
			if err == io.EOF {
				return nil, errNoSuchPart
			} else if err != nil {
				return nil, err
			}

			if j == n {
				e = p
				break
			}
		}
	}

	// Then, write the requested data to a buffer
	b := new(bytes.Buffer)

	header := e.Header
	if section.Fields != nil {
		// Copy header so we will not change message.Entity passed to us.
		header.Header = e.Header.Copy()

		if section.NotFields {
			for _, fieldName := range section.Fields {
				header.Del(fieldName)
			}
		} else {
			fieldsMap := make(map[string]struct{}, len(section.Fields))
			for _, field := range section.Fields {
				fieldsMap[textproto.CanonicalMIMEHeaderKey(field)] = struct{}{}
			}

			for field := header.Fields(); field.Next(); {
				if _, ok := fieldsMap[field.Key()]; !ok {
					field.Del()
				}
			}
		}
	}

	// Write the header
	mw, err := message.CreateWriter(b, header)
	if err != nil {
		return nil, err
	}
	defer mw.Close()

	switch section.Specifier {
	case imap.TextSpecifier:
		// The header hasn't been requested. Discard it.
		b.Reset()
	case imap.EntireSpecifier:
		if len(section.Path) > 0 {
			// When selecting a specific part by index, IMAP servers
			// return only the text, not the associated MIME header.
			b.Reset()
		}
	}

	// Write the body, if requested
	switch section.Specifier {
	case imap.EntireSpecifier, imap.TextSpecifier:
		if _, err := io.Copy(mw, e.Body); err != nil {
			return nil, err
		}
	}

	var l imap.Literal = b
	if section.Partial != nil {
		l = bytes.NewReader(section.ExtractPartial(b.Bytes()))
	}
	return l, nil
}
