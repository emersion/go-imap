package common

import (
	"time"
)

type Message struct {
	Fields map[string]interface{}
}

// TODO
func (m *Message) Body() (lit *Literal) { return }
func (m *Message) BodyStructure() (s *BodyStructure) { return }
func (m *Message) Envelope() (e *Envelope) { return }
func (m *Message) Flags() (flags []string) { return }
func (m *Message) InternalDate() (t *time.Time) { return }
func (m *Message) Size() (size int) { return }
func (m *Message) Uid() (uid int) { return }

type Envelope struct {
	Date *time.Time
	Subject string
	From []*Address
	Sender []*Address
	ReplyTo []*Address
	To []*Address
	Cc []*Address
	Bcc []*Address
	InReplyTo string
	MessageId string
}

type Address struct {
	PersonalName string
	AtDomainList string
	MailboxName string
	HostName string
}

type BodyStructure struct {
	MimeType string
	MimeSubType string
	Params map[string]string
	Id string
	Description string
	Encoding string
	Size int
}
