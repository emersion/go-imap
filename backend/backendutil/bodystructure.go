package backendutil

import (
	"io"
	"mime"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/textproto"
)

// FetchBodyStructure computes a message's body structure from its content.
func FetchBodyStructure(header textproto.Header, body io.Reader, extended bool) (*imap.BodyStructure, error) {
	bs := new(imap.BodyStructure)

	mediaType, mediaParams, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err == nil {
		typeParts := strings.SplitN(mediaType, "/", 2)
		bs.MIMEType = typeParts[0]
		if len(typeParts) == 2 {
			bs.MIMESubType = typeParts[1]
		}
		bs.Params = mediaParams
	} else {
		bs.MIMEType = "text"
		bs.MIMESubType = "plain"
	}

	bs.Id = header.Get("Content-Id")
	bs.Description = header.Get("Content-Description")
	bs.Encoding = header.Get("Content-Transfer-Encoding")
	// TODO: bs.Size

	if mr := multipartReader(header, body); mr != nil {
		var parts []*imap.BodyStructure
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			}

			pbs, err := FetchBodyStructure(p.Header, p, extended)
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

		bs.Disposition, bs.DispositionParams, _ = mime.ParseMediaType(header.Get("Content-Disposition"))

		// TODO: bs.Language, bs.Location
		// TODO: bs.MD5
	}

	return bs, nil
}
