package memory

import (
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

			metadata.Body[section] = common.NewLiteral(m.body)
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
