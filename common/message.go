package common

import (
	"errors"
	"time"
)

// Layouts suitable for passing to time.Parse.
var dateLayouts []string

func init() {
	// Generate layouts based on RFC 5322, section 3.3.
	dows := []string{"", "Mon, "}   // day-of-week
	days := []string{"2", "02"}     // day = 1*2DIGIT
	years := []string{"2006", "06"} // year = 4*DIGIT / 2*DIGIT
	seconds := []string{":05", ""}  // second
	// "-0700 (MST)" is not in RFC 5322, but is common.
	zones := []string{"-0700", "MST", "-0700 (MST)"} // zone = (("+" / "-") 4DIGIT) / "GMT" / ...

	for _, dow := range dows {
		for _, day := range days {
			for _, year := range years {
				for _, second := range seconds {
					for _, zone := range zones {
						s := dow + day + " Jan " + year + " 15:04" + second + " " + zone
						dateLayouts = append(dateLayouts, s)
					}
				}
			}
		}
	}
}

// Parse an IMAP date.
// Borrowed from https://golang.org/pkg/net/mail/#Header.Date
func ParseDate(date string) (*time.Time, error) {
	for _, layout := range dateLayouts {
		t, err := time.Parse(layout, date)
		if err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("Cannot parse date")
}

func ParseAddress(fields []interface{}) *Address {
	if len(fields) < 4 {
		return nil
	}

	addr := &Address{}

	if f, ok := fields[0].(string); ok {
		addr.PersonalName = f
	}
	if f, ok := fields[1].(string); ok {
		addr.AtDomainList = f
	}
	if f, ok := fields[2].(string); ok {
		addr.MailboxName = f
	}
	if f, ok := fields[3].(string); ok {
		addr.HostName = f
	}

	return addr
}

func ParseAddressList(fields []interface{}) (addrs []*Address) {
	for _, f := range fields {
		if addrFields, ok := f.([]interface{}); ok {
			if addr := ParseAddress(addrFields); addr != nil {
				addrs = append(addrs, addr)
			}
		}
	}
	return
}

type Message struct {
	Id uint32
	Fields map[string]interface{}
}

func (m *Message) Envelope() *Envelope {
	fields, ok := m.Fields["ENVELOPE"].([]interface{})
	if !ok || len(fields) < 10 {
		return nil
	}

	e := &Envelope{}

	if date, ok := fields[0].(string); ok {
		e.Date, _ = ParseDate(date)
	}
	if subject, ok := fields[1].(string); ok {
		e.Subject = subject
	}
	if from, ok := fields[2].([]interface{}); ok {
		e.From = ParseAddressList(from)
	}
	if sender, ok := fields[3].([]interface{}); ok {
		e.Sender = ParseAddressList(sender)
	}
	if replyTo, ok := fields[4].([]interface{}); ok {
		e.ReplyTo = ParseAddressList(replyTo)
	}
	if to, ok := fields[5].([]interface{}); ok {
		e.To = ParseAddressList(to)
	}
	if cc, ok := fields[6].([]interface{}); ok {
		e.Cc = ParseAddressList(cc)
	}
	if bcc, ok := fields[7].([]interface{}); ok {
		e.Bcc = ParseAddressList(bcc)
	}
	if inReplyTo, ok := fields[8].(string); ok {
		e.InReplyTo = inReplyTo
	}
	if msgId, ok := fields[9].(string); ok {
		e.MessageId = msgId
	}

	return e
}

// TODO
func (m *Message) Body() (lit *Literal) { return }
func (m *Message) BodyStructure() (s *BodyStructure) { return }
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
