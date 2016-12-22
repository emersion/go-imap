package memory

import (
	"bytes"

	"github.com/emersion/go-imap"
)

type Message struct {
	*imap.Message

	body []byte
}

func (m *Message) Metadata(items []string) (metadata *imap.Message) {
	metadata = imap.NewMessage(0, items)

	for _, item := range items {
		switch item {
		case imap.EnvelopeMsgAttr:
			metadata.Envelope = m.Envelope
		case imap.BodyMsgAttr, imap.BodyStructureMsgAttr:
			metadata.BodyStructure = m.BodyStructure
		case imap.FlagsMsgAttr:
			metadata.Flags = m.Flags
		case imap.InternalDateMsgAttr:
			metadata.InternalDate = m.InternalDate
		case imap.SizeMsgAttr:
			metadata.Size = m.Size
		case imap.UidMsgAttr:
			metadata.Uid = m.Uid
		default:
			section, err := imap.NewBodySectionName(item)
			if err != nil {
				break
			}

			var body []byte
			if len(section.Path) == 0 {
				if section.Specifier == "" {
					body = m.body
				} else {
					sep := []byte("\n\n")
					parts := bytes.SplitN(m.body, sep, 2)
					if len(parts) == 1 {
						parts = [][]byte{nil, parts[0]}
					}

					if section.Specifier == imap.HeaderSpecifier {
						body = parts[0]
						body = append(body, sep...)
					}
					if section.Specifier == imap.TextSpecifier {
						body = parts[1]
					}
				}
			}

			// If part doesn't exist, set the literal to nil
			var literal imap.Literal
			if body != nil {
				literal = bytes.NewBuffer(section.ExtractPartial(body))
			}
			metadata.Body[section] = literal
		}
	}

	return
}

func (m *Message) hasFlag(flag string) bool {
	for _, f := range m.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

func (m *Message) Matches(criteria *imap.SearchCriteria) bool {
	// TODO
	if criteria.SeqSet != nil && !criteria.SeqSet.Contains(m.SeqNum) {
		return false
	}
	if criteria.Deleted && !m.hasFlag(imap.DeletedFlag) {
		return false
	}
	return true
}
