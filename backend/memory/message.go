package memory

import (
	"bytes"
	"log"

	"github.com/emersion/imap/common"
)

type Message struct {
	*common.Message

	body []byte
}

func (m *Message) Metadata(items []string) (metadata *common.Message) {
	metadata = &common.Message{
		Body: map[*common.BodySectionName]*common.Literal{},
	}

	for _, item := range items {
		switch item {
		case "ENVELOPE":
			metadata.Envelope = m.Envelope
		case "BODYSTRUCTURE", "BODY":
			metadata.BodyStructure = m.BodyStructure
		case "FLAGS":
			metadata.Flags = m.Flags
		case "INTERNALDATE":
			metadata.InternalDate = m.InternalDate
		case "RFC822.SIZE":
			metadata.Size = m.Size
		case "UID":
			metadata.Uid = m.Uid
		default:
			section, err := common.NewBodySectionName(item)
			item = ""
			if err != nil {
				break
			}
log.Println(section.Specifier)
			var body []byte
			if section.Specifier == "" {
				body = m.body
			} else {
				sep := []byte("\n\n")
				parts := bytes.SplitN(m.body, sep, 2)
				if len(parts) == 1 {
					parts = [][]byte{nil, parts[0]}
				}

				if section.Specifier == "HEADER" {
					body = parts[0]
					body = append(body, sep...)
				}
				if section.Specifier == "TEXT" {
					body = parts[1]
				}
			}

			metadata.Body[section] = common.NewLiteral(section.ExtractPartial(body))
		}

		if item != "" {
			metadata.Items = append(metadata.Items, item)
		}
	}

	return
}

func (m *Message) Matches(criteria *common.SearchCriteria) bool {
	return true // TODO
}
