package backendutil

import (
	"io"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
)

// FetchBodyStructure computes a message's body structure from its content.
func FetchBodyStructure(e *message.Entity, extended bool) (*imap.BodyStructure, error) {
	bs := new(imap.BodyStructure)

	mediaType, mediaParams, _ := e.Header.ContentType()
	typeParts := strings.SplitN(mediaType, "/", 2)
	bs.MIMEType = typeParts[0]
	if len(typeParts) == 2 {
		bs.MIMESubType = typeParts[1]
	}
	bs.Params = mediaParams

	bs.Id = e.Header.Get("Content-Id")
	bs.Description = e.Header.Get("Content-Description")
	bs.Encoding = e.Header.Get("Content-Encoding")
	// TODO: bs.Size

	if mr := e.MultipartReader(); mr != nil {
		var parts []*imap.BodyStructure
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			}

			pbs, err := FetchBodyStructure(p, extended)
			if err != nil {
				return nil, err
			}
			parts = append(parts, pbs)
		}
		bs.Parts = parts
	}

	// TODO: bs.Envelope, bs.BodyStructure
	// TODO: bs.Lines

	if extended {
		bs.Extended = true

		bs.Disposition, bs.DispositionParams, _ = e.Header.ContentDisposition()

		// TODO: bs.Language, bs.Location
		// TODO: bs.MD5
	}

	return bs, nil
}
