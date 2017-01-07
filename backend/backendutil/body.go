package backendutil

import (
	"bytes"
	"errors"
	"io"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
)

var errNoSuchPart = errors.New("backendutil: no such message body part")

func FetchBodySection(e *message.Entity, section *imap.BodySectionName) (imap.Literal, error) {
	// First, find the requested part using the provided path
	for i := len(section.Path) - 1; i >= 0; i-- {
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

	// Write the header, if requested
	var w io.Writer
	switch section.Specifier {
	case imap.EntireSpecifier, imap.HeaderSpecifier, imap.MimeSpecifier:
		mw, err := message.CreateWriter(b, e.Header)
		if err != nil {
			return nil, err
		}
		defer mw.Close()
		w = mw
	default: // imap.TextSpecifier
		w = b
	}

	// Write the body, if requested
	switch section.Specifier {
	case imap.EntireSpecifier, imap.TextSpecifier:
		if _, err := io.Copy(w, e.Body); err != nil {
			return nil, err
		}
	}

	return b, nil
}
