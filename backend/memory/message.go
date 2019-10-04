package memory

import (
	"bufio"
	"bytes"
	"io"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/textproto"
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

func (m *Message) headerAndBody() (textproto.Header, io.Reader, error) {
	body := bufio.NewReader(bytes.NewReader(m.Body))
	hdr, err := textproto.ReadHeader(body)
	return hdr, body, err
}

func (m *Message) Fetch(seqNum uint32, items []imap.FetchItem) (*imap.Message, error) {
	fetched := imap.NewMessage(seqNum, items)
	for _, item := range items {
		switch item {
		case imap.FetchEnvelope:
			hdr, _, err := m.headerAndBody()
			if err != nil {
				return nil, err
			}
			if envelope, err := backendutil.FetchEnvelope(hdr); err != nil {
				return nil, err
			} else {
				fetched.Envelope = envelope
			}
		case imap.FetchBody, imap.FetchBodyStructure:
			hdr, body, err := m.headerAndBody()
			if err != nil {
				return nil, err
			}
			if body, err := backendutil.FetchBodyStructure(hdr, body, item == imap.FetchBodyStructure); err != nil {
				return nil, err
			} else {
				fetched.BodyStructure = body
			}
		case imap.FetchFlags:
			// Copy flags, don't return reference to message's flags slice.
			flags := append(m.Flags[:0:0], m.Flags...)
			fetched.Flags = flags
		case imap.FetchInternalDate:
			fetched.InternalDate = m.Date
		case imap.FetchRFC822Size:
			fetched.Size = m.Size
		case imap.FetchUid:
			fetched.Uid = m.Uid
		default:
			section, err := imap.ParseBodySectionName(item)
			if err != nil {
				break
			}

			body := bufio.NewReader(bytes.NewReader(m.Body))
			hdr, err := textproto.ReadHeader(body)
			if err != nil {
				return nil, err
			}

			l, _ := backendutil.FetchBodySection(hdr, body, section)
			fetched.Body[section] = l
		}
	}

	return fetched, nil
}

func (m *Message) Match(seqNum uint32, c *imap.SearchCriteria) (bool, error) {
	e, _ := m.entity()
	return backendutil.Match(e, seqNum, m.Uid, m.Date, m.Flags, c)
}
