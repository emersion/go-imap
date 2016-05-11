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
		Body: map[string]*common.Literal{},
	}

	for _, item := range items {
		ok := true
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
			if err != nil {
				ok = false
				break
			}

			//metadata.Body[item] = m.Body[item]
			metadata.Body[section.Resp().String()] = common.NewLiteral(m.body)
		}

		if ok {
			metadata.Items = append(metadata.Items, item)
		}
	}

	return
}

func (m *Message) Matches(criteria *common.SearchCriteria) bool {
	return true // TODO
}
