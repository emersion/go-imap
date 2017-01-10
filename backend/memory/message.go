package memory

import (
	"bytes"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-message"
)

type Message struct {
	Uid   uint32
	Date  time.Time
	Size  uint32
	Flags []string
	Body  []byte
}

func (m *Message) entity() (*message.Entity, error) {
	return message.Read(bytes.NewReader(m.Body))
}

func (m *Message) Fetch(seqNum uint32, items []string) (*imap.Message, error) {
	fetched := imap.NewMessage(seqNum, items)
	for _, item := range items {
		switch item {
		case imap.EnvelopeMsgAttr:
			e, _ := m.entity()
			fetched.Envelope, _ = backendutil.FetchEnvelope(e.Header)
		case imap.BodyMsgAttr, imap.BodyStructureMsgAttr:
			e, _ := m.entity()
			fetched.BodyStructure, _ = backendutil.FetchBodyStructure(e, item == imap.BodyStructureMsgAttr)
		case imap.FlagsMsgAttr:
			fetched.Flags = m.Flags
		case imap.InternalDateMsgAttr:
			fetched.InternalDate = m.Date
		case imap.SizeMsgAttr:
			fetched.Size = m.Size
		case imap.UidMsgAttr:
			fetched.Uid = m.Uid
		default:
			section, err := imap.NewBodySectionName(item)
			if err != nil {
				break
			}

			e, _ := m.entity()
			l, _ := backendutil.FetchBodySection(e, section)
			fetched.Body[section] = l
		}
	}

	return fetched, nil
}

func (m *Message) Match(seqNum uint32, c *imap.SearchCriteria) (bool, error) {
	if !backendutil.MatchSeqNumAndUid(seqNum, m.Uid, c) {
		return false, nil
	}
	if !backendutil.MatchDate(m.Date, c) {
		return false, nil
	}
	if !backendutil.MatchFlags(m.Flags, c) {
		return false, nil
	}

	e, _ := m.entity()
	return backendutil.Match(e, c)
}
