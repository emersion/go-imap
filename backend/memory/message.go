package memory

import (
	"github.com/emersion/imap/common"
)

type Message struct {
	metadata *common.Message
}

func (m *Message) Metadata(items []string) (metadata *common.Message) {
	metadata = &common.Message{Items: items}

	for _, item := range items {
		switch item {
		case "ENVELOPE":
			metadata.Envelope = m.metadata.Envelope
		case "BODYSTRUCTURE", "BODY":
			metadata.BodyStructure = m.metadata.BodyStructure
		case "FLAGS":
			metadata.Flags = m.metadata.Flags
		case "INTERNALDATE":
			metadata.InternalDate = m.metadata.InternalDate
		case "RFC822.SIZE":
			metadata.Size = m.metadata.Size
		case "UID":
			metadata.Uid = m.metadata.Uid
		}
	}

	return
}
