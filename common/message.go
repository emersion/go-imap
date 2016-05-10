package common

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
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

// A message.
type Message struct {
	// The message identifier. Can be either a sequence number or a UID, depending
	// of how this message has been retrieved.
	Id uint32
	// The message items that are currently filled in.
	Items []string

	// The message envelope.
	Envelope *Envelope
	// The message body parts.
	Body map[string]*Literal
	// The message body structure (either BODYSTRUCTURE or BODY).
	BodyStructure *BodyStructure
	// The message flags.
	Flags []string
	// The date the message was received by th server.
	InternalDate *time.Time
	// The message size.
	Size uint32
	// The message UID.
	Uid uint32
}

// Parse a message from fields.
func (m *Message) Parse(fields []interface{}) error {
	m.Items = nil
	m.Body = map[string]*Literal{}

	var key string
	for i, f := range fields {
		if i % 2 == 0 { // It's a key
			var ok bool
			if key, ok = f.(string); !ok {
				return errors.New("Key is not a string")
			}
		} else { // It's a value
			key = strings.ToUpper(key)
			m.Items = append(m.Items, key)

			switch key {
			case "ENVELOPE":
				env, ok := f.([]interface{})
				if !ok {
					return errors.New("ENVELOPE is not a list")
				}

				m.Envelope = &Envelope{}
				if err := m.Envelope.Parse(env); err != nil {
					return err
				}
			case "BODYSTRUCTURE", "BODY":
				bs, ok := f.([]interface{})
				if !ok {
					return errors.New("BODYSTRUCTURE is not a list")
				}

				m.BodyStructure = &BodyStructure{}
				if err := m.BodyStructure.Parse(bs); err != nil {
					return err
				}
			case "FLAGS":
				flags, ok := f.([]interface{})
				if !ok {
					return errors.New("FLAGS is not a list")
				}

				m.Flags = make([]string, len(flags))
				for i, flag := range flags {
					m.Flags[i], _ = flag.(string)
				}
			case "INTERNALDATE":
				date, _ := f.(string)
				m.InternalDate, _ = ParseDate(date)
			case "RFC822.SIZE":
				m.Size, _ = ParseNumber(f)
			case "UID":
				m.Uid, _ = ParseNumber(f)
			default:
				// Likely to be a section of the body contents
				literal, ok := f.(*Literal)
				if !ok {
					break
				}
				m.Body[key] = literal
			}
		}
	}
	return nil
}

func (m *Message) Format() (fields []interface{}) {
	for _, item := range m.Items {
		var key, value interface{}
		key = item

		switch strings.ToUpper(item) {
		case "ENVELOPE":
			value = m.Envelope.Format()
		case "BODYSTRUCTURE", "BODY":
			value = m.BodyStructure.Format()
		case "FLAGS":
			flags := make([]interface{}, len(m.Flags))
			for i, v := range m.Flags {
				flags[i] = v
			}
			value = flags
		case "INTERNALDATE":
			value = m.InternalDate
		case "RFC822.SIZE":
			value = m.Size
		case "UID":
			value = m.Uid
		default:
			var ok bool
			if value, ok = m.Body[item]; ok {
				key = &BodySectionName{value: item}
			} else {
				key = nil
			}
		}

		if key == nil {
			continue
		}

		fields = append(fields, key, value)
	}

	return
}

// A body section name.
// See RFC 3501 page 55.
type BodySectionName struct {
	// If set to true, do not implicitely set the \Seen flag.
	Peek bool
	// The list of parts requested in the section.
	Parts map[string]interface{}
	// The substring of the section requested. The first value is the position of
	// the first desired octet and the second value is the maximum number of
	// octets desired.
	Partial []uint32

	value string
}

func (section *BodySectionName) parse(s string) (err error) {
	section.value = s

	if s == "RFC822" {
		s = "BODY[]"
	}
	if s == "RFC822.HEADER" {
		s = "BODY.PEEK[HEADER]"
	}
	if s == "RFC822.TEXT" {
		s = "BODY[TEXT]"
	}

	partsStart := strings.Index(s, "[")
	if partsStart == -1 {
		return errors.New("Invalid body section name: must contain an open bracket")
	}

	partsEnd := strings.LastIndex(s, "]")
	if partsEnd == -1 {
		return errors.New("Invalid body section name: must contain a close bracket")
	}

	name := s[:partsStart]
	parts := s[partsStart+1:partsEnd]
	partial := s[partsEnd+1:]

	if name == "BODY.PEEK" {
		section.Peek = true
	} else if name != "BODY" {
		return errors.New("Invalid body section name")
	}

	section.Parts = map[string]interface{}{}

	var b *bytes.Buffer
	for _, part := range strings.Split(parts, ",") {
		b = bytes.NewBufferString(part)
		r := NewReader(b)

		var fields []interface{}
		if fields, err = r.ReadFields(); err != nil {
			return err
		}
		if len(fields) < 0 {
			return errors.New("Invalid body section name: empty part")
		}

		name, ok := fields[0].(string)
		if !ok {
			return errors.New("Invalid body section name: part name must be a string")
		}

		var value interface{}
		if len(fields) > 1 {
			value = fields[1]
		}

		section.Parts[name] = value
	}

	if len(partial) > 0 {
		if !strings.HasPrefix(partial, "<") || !strings.HasSuffix(partial, ">") {
			return errors.New("Invalid body section name: invalid partial")
		}
		partial = partial[1:len(partial)-1]

		partialParts := strings.SplitN(partial, ",", 2)
		if len(partialParts) < 2 {
			return errors.New("Invalid body section name: partial must have two fields")
		}

		var from, to uint64
		if from, err = strconv.ParseUint(partialParts[0], 10, 32); err != nil {
			return errors.New("Invalid body section name: " + err.Error())
		}
		if to, err = strconv.ParseUint(partialParts[1], 10, 32); err != nil {
			return errors.New("Invalid body section name: " + err.Error())
		}
		section.Partial = []uint32{uint32(from), uint32(to)}
	}

	return nil
}

func (section *BodySectionName) String() (s string) {
	if section.value != "" {
		return section.value
	}

	s = "BODY"
	if section.Peek {
		s += ".PEEK"
	}
	s += "["

	var b bytes.Buffer
	w := NewWriter(&b)

	first := true
	for name, arg := range section.Parts {
		fields := []interface{}{name}
		if arg != nil {
			fields = append(fields, arg)
		}

		w.WriteFields(fields)

		s += b.String()
		b.Reset()

		if first {
			first = false
		} else {
			s += ","
		}
	}

	s += "]"
	if len(section.Partial) > 0 {
		s += "<"
		s += strconv.FormatUint(uint64(section.Partial[0]), 10)

		if len(section.Partial) > 1 {
			s += ","
			s += strconv.FormatUint(uint64(section.Partial[0]), 10)
		}

		s += ">"
	}

	return
}

// Parse a body section name.
func NewBodySectionName(s string) (section *BodySectionName, err error) {
	section = &BodySectionName{}
	err = section.parse(s)
	return
}

// A message envelope, ie. message metadata from its headers.
// See RFC 3501 page 77.
type Envelope struct {
	// The message date.
	Date *time.Time
	// The message subject.
	Subject string
	// The From header addresses.
	From []*Address
	// The message senders.
	Sender []*Address
	// The Reply-To header addresses.
	ReplyTo []*Address
	// The To header addresses.
	To []*Address
	// The Cc header addresses.
	Cc []*Address
	// The Bcc header addresses.
	Bcc []*Address
	// The In-Reply-To header. Contains the parent Message-Id.
	InReplyTo string
	// The Message-Id header.
	MessageId string
}

// Parse an envelope from fields.
func (e *Envelope) Parse(fields []interface{}) error {
	if len(fields) < 10 {
		return errors.New("ENVELOPE doesn't contain 10 fields")
	}

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

	return nil
}

// Format an envelope to fields.
func (e *Envelope) Format() (fields []interface{}) {
	return []interface{}{
		e.Date,
		e.Subject,
		FormatAddressList(e.From),
		FormatAddressList(e.Sender),
		FormatAddressList(e.ReplyTo),
		FormatAddressList(e.To),
		FormatAddressList(e.Cc),
		FormatAddressList(e.Bcc),
		e.InReplyTo,
		e.MessageId,
	}
}

// An address.
type Address struct {
	// The personal name.
	PersonalName string
	// The SMTP at-domain-list (source route).
	AtDomainList string
	// The mailbox name.
	MailboxName string
	// The host name.
	HostName string
}

func (addr *Address) String() string {
	return addr.MailboxName + "@" + addr.HostName
}

// Parse an address from fields.
func (addr *Address) Parse(fields []interface{}) error {
	if len(fields) < 4 {
		return errors.New("Address doesn't contain 4 fields")
	}

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

	return nil
}

// Format an address to fields.
func (addr *Address) Format() []interface{} {
	return []interface{}{
		addr.PersonalName,
		addr.AtDomainList,
		addr.MailboxName,
		addr.HostName,
	}
}

// Parse an address list from fields.
func ParseAddressList(fields []interface{}) (addrs []*Address) {
	addrs = make([]*Address, len(fields))

	for i, f := range fields {
		if addrFields, ok := f.([]interface{}); ok {
			addr := &Address{}
			if err := addr.Parse(addrFields); err == nil {
				addrs[i] = addr
			}
		}
	}

	return
}

// Format an address list to fields.
func FormatAddressList(addrs []*Address) (fields []interface{}) {
	fields = make([]interface{}, len(addrs))

	for i, addr := range addrs {
		fields[i] = addr.Format()
	}

	return
}

// A body structure.
// See RFC 3501 page 74.
type BodyStructure struct {
	// Basic fields

	// The MIME type.
	MimeType string
	// The MIME subtype.
	MimeSubType string
	// The MIME parameters.
	Params map[string]string

	// The Content-Id header.
	Id string
	// The Content-Description header.
	Description string
	// The Content-Encoding header.
	Encoding string
	// The Content-Length header.
	Size uint32

	// Type-specific fields

	// The children parts, if multipart.
	Parts []*BodyStructure
	// The envelope, if message/rfc822.
	Envelope *Envelope
	// The body structure, if message/rfc822.
	BodyStructure *BodyStructure
	// The number of lines, if text or message/rfc822.
	Lines uint32

	// Extension data

	// The Content-Disposition header.
	Disposition string
	// The Content-Language header, if multipart.
	Language []string
	// The content URI, if multipart.
	Location []string

	// The MD5 checksum.
	Md5 string
}

func ParseParamList(fields []interface{}) (params map[string]string, err error) {
	params = map[string]string{}

	var key string
	for i, f := range fields {
		p, ok := f.(string)
		if !ok {
			err = errors.New("Parameter list contains a non-string")
			return
		}

		if i % 2 == 0 {
			key = p
		} else {
			params[key] = p
		}
	}

	return
}

func FormatParamList(params map[string]string) (fields []interface{}) {
	for key, value := range params {
		fields = append(fields, key, value)
	}
	return
}

func ParseStringList(fields []interface{}) ([]string, error) {
	list := make([]string, len(fields))
	for i, f := range fields {
		var ok bool
		if list[i], ok = f.(string); !ok {
			return nil, errors.New("String list contains a non-string")
		}
	}
	return list, nil
}

func FormatStringList(list []string) (fields []interface{}) {
	fields = make([]interface{}, len(list))
	for i, v := range list {
		fields[i] = v
	}
	return
}

func (bs *BodyStructure) Parse(fields []interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	switch fields[0].(type) {
	case []interface{}: // A multipart body part
		bs.MimeType = "multipart"

		end := 0
		for i, fi := range fields {
			switch f := fi.(type) {
			case []interface{}: // A part
				part := &BodyStructure{}
				if err := part.Parse(f); err != nil {
					return err
				}
				bs.Parts = append(bs.Parts, part)
			case string:
				end = i
			}

			if end > 0 {
				break
			}
		}

		bs.MimeSubType, _ = fields[end].(string)

		if len(fields) - end + 1 >= 4 { // Contains extension data
			params, _ := fields[end].([]interface{})
			bs.Params, _ = ParseParamList(params)

			bs.Disposition, _ = fields[end+1].(string)

			switch langs := fields[end+2].(type) {
			case string:
				bs.Language = []string{langs}
			case []interface{}:
				bs.Language, _ = ParseStringList(langs)
			}

			location, _ := fields[end+3].([]interface{})
			bs.Location, _ = ParseStringList(location)
		}
	case string: // A non-multipart body part
		if len(fields) < 7 {
			return errors.New("Non-multipart body part doesn't have 7 fields")
		}

		bs.MimeType, _ = fields[0].(string)
		bs.MimeSubType, _ = fields[1].(string)

		params, _ := fields[2].([]interface{})
		bs.Params, _ = ParseParamList(params)

		bs.Id, _ = fields[3].(string)
		bs.Description, _ = fields[4].(string)
		bs.Encoding, _ = fields[5].(string)
		bs.Size, _ = ParseNumber(fields[6])

		end := 7

		// Type-specific fields
		if bs.MimeType == "message" && bs.MimeSubType == "rfc822" {
			envelope, _ := fields[end].([]interface{})
			bs.Envelope = &Envelope{}
			bs.Envelope.Parse(envelope)

			structure, _ := fields[end+1].([]interface{})
			bs.BodyStructure = &BodyStructure{}
			bs.BodyStructure.Parse(structure)

			bs.Lines, _ = ParseNumber(fields[end+2])

			end += 3
		}
		if bs.MimeType == "text" {
			bs.Lines, _ = ParseNumber(fields[end])
			end++
		}

		if len(fields) - end + 1 >= 4 { // Contains extension data
			bs.Md5, _ = fields[end].(string)
			bs.Disposition, _ = fields[end+1].(string)

			switch langs := fields[end+2].(type) {
			case string:
				bs.Language = []string{langs}
			case []interface{}:
				bs.Language, _ = ParseStringList(langs)
			}

			location, _ := fields[end+3].([]interface{})
			bs.Location, _ = ParseStringList(location)
		}
	}

	return nil
}

func (bs *BodyStructure) Format() (fields []interface{}) {
	if bs.MimeType == "multipart" {
		for _, part := range bs.Parts {
			fields = append(fields, part.Format())
		}

		fields = append(fields, bs.MimeSubType)

		if bs.Params != nil || bs.Disposition != "" || len(bs.Language) > 0 || len(bs.Location) > 0 {
			fields = append(fields, FormatParamList(bs.Params), bs.Disposition,
				FormatStringList(bs.Language), FormatStringList(bs.Location))
		}
	} else {
		fields = []interface{}{
			bs.MimeType,
			bs.MimeSubType,
			FormatParamList(bs.Params),
			bs.Id,
			bs.Description,
			bs.Encoding,
			bs.Size,
		}

		// Type-specific fields
		if bs.MimeType == "message" && bs.MimeSubType == "rfc822" {
			fields = append(fields, bs.Envelope, bs.BodyStructure, bs.Lines)
		}
		if bs.MimeType == "text" {
			fields = append(fields, bs.Lines)
		}

		// Extension data
		if bs.Md5 != "" || bs.Disposition != "" || len(bs.Language) > 0 || len(bs.Location) > 0 {
			fields = append(fields, bs.Md5, bs.Disposition,
				FormatStringList(bs.Language), FormatStringList(bs.Location))
		}
	}

	return
}
