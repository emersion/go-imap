package imap

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Message flags, defined in RFC 3501 section 2.3.2.
const (
	SeenFlag     = "\\Seen"
	AnsweredFlag = "\\Answered"
	FlaggedFlag  = "\\Flagged"
	DeletedFlag  = "\\Deleted"
	DraftFlag    = "\\Draft"
	RecentFlag   = "\\Recent"
)

var flags = []string{
	SeenFlag,
	AnsweredFlag,
	FlaggedFlag,
	DeletedFlag,
	DraftFlag,
	RecentFlag,
}

// Message attributes that can be fetched, defined in RFC 3501 section 6.4.5.
// Attributes that fetches the message contents are defined with
// BodySectionName.
const (
	// Non-extensible form of BODYSTRUCTURE.
	BodyMsgAttr = "BODY"
	// MIME body structure of the message.
	BodyStructureMsgAttr = "BODYSTRUCTURE"
	// The envelope structure of the message.
	EnvelopeMsgAttr = "ENVELOPE"
	// The flags that are set for the message.
	FlagsMsgAttr = "FLAGS"
	// The internal date of the message.
	InternalDateMsgAttr = "INTERNALDATE"
	// The RFC 822 size of the message.
	SizeMsgAttr = "RFC822.SIZE"
	// The unique identifier for the message.
	UidMsgAttr = "UID"
)

// Part specifiers described in RFC 3501 page 55.
const (
	// Refers to the entire part, including headers.
	EntireSpecifier = ""
	// Refers to the header of the part. Must include the final CRLF delimiting
	// the header and the body.
	HeaderSpecifier = "HEADER"
	// Refers to the text body of the part, omitting the header.
	TextSpecifier = "TEXT"
	// Refers to the MIME Internet Message Body header.  Must include the final
	// CRLF delimiting the header and the body.
	MimeSpecifier = "MIME"
)

// Date and time formats used as examples in RFC 3501 section 8.
const (
	// The date format described in RFC 2822 section 3.3.
	EnvelopeDateTimeFormat = "Mon, 02 Jan 2006 15:04:05 -0700"
	// Described in RFC 1730 on page 55.
	DateFormat = "2-Jan-2006"
	// Described in RFC 1730 on page 55.
	DateTimeFormat = "2-Jan-2006 15:04:05 -0700"
)

// A time.Time with a specific layout
type timeLayout struct {
	date   time.Time
	layout string
}

// Returns the canonical form of a flag. Flags are case-insensitive.
//
// If the flag is defined in RFC 3501, it returns the flag with the case of the
// RFC. Otherwise, it returns the lowercase version of the flag.
func CanonicalFlag(flag string) string {
	flag = strings.ToLower(flag)
	for _, f := range flags {
		if strings.ToLower(f) == flag {
			return f
		}
	}
	return flag
}

// A message.
type Message struct {
	// The message sequence number. It must be greater than or equal to 1.
	SeqNum uint32
	// The message items that are currently filled in.
	Items []string
	// The message body sections.
	Body map[*BodySectionName]*Literal

	// The message envelope.
	Envelope *Envelope
	// The message body structure (either BODYSTRUCTURE or BODY).
	BodyStructure *BodyStructure
	// The message flags.
	Flags []string
	// The date the message was received by the server.
	InternalDate time.Time
	// The message size.
	Size uint32
	// The message unique identifier. It must be greater than or equal to 1.
	Uid uint32
}

// Create a new empty message.
func NewMessage() *Message {
	return &Message{Body: map[*BodySectionName]*Literal{}}
}

// Parse a message from fields.
func (m *Message) Parse(fields []interface{}) error {
	m.Items = nil
	m.Body = map[*BodySectionName]*Literal{}

	var key string
	for i, f := range fields {
		if i%2 == 0 { // It's a key
			var ok bool
			if key, ok = f.(string); !ok {
				return errors.New("Key is not a string")
			}
		} else { // It's a value
			item := strings.ToUpper(key)

			switch item {
			case BodyMsgAttr, BodyStructureMsgAttr:
				bs, ok := f.([]interface{})
				if !ok {
					return errors.New("BODYSTRUCTURE is not a list")
				}

				m.BodyStructure = &BodyStructure{Extended: item == BodyStructureMsgAttr}
				if err := m.BodyStructure.Parse(bs); err != nil {
					return err
				}
			case EnvelopeMsgAttr:
				env, ok := f.([]interface{})
				if !ok {
					return errors.New("ENVELOPE is not a list")
				}

				m.Envelope = &Envelope{}
				if err := m.Envelope.Parse(env); err != nil {
					return err
				}
			case FlagsMsgAttr:
				flags, ok := f.([]interface{})
				if !ok {
					return errors.New("FLAGS is not a list")
				}

				m.Flags = make([]string, len(flags))
				for i, flag := range flags {
					s, _ := flag.(string)
					m.Flags[i] = CanonicalFlag(s)
				}
			case InternalDateMsgAttr:
				date, _ := f.(string)
				m.InternalDate, _ = time.Parse(DateTimeFormat, date)
			case SizeMsgAttr:
				m.Size, _ = ParseNumber(f)
			case UidMsgAttr:
				m.Uid, _ = ParseNumber(f)
			default:
				// Likely to be a section of the body
				// First check that the section name is correct
				section, err := NewBodySectionName(item)
				if err != nil {
					return err
				}

				// Then check that the value is a correct literal
				literal, ok := f.(*Literal)
				if !ok {
					break
				}

				m.Body[section] = literal

				// Do not include this in the list of items
				item = ""
			}

			if item != "" {
				m.Items = append(m.Items, item)
			}
		}
	}
	return nil
}

func (m *Message) Format() (fields []interface{}) {
	for _, item := range m.Items {
		item = strings.ToUpper(item)

		ok := true
		var value interface{}
		switch item {
		case BodyMsgAttr, BodyStructureMsgAttr:
			// Extension data is only returned with the BODYSTRUCTURE fetch
			m.BodyStructure.Extended = item == BodyStructureMsgAttr
			value = m.BodyStructure.Format()
		case EnvelopeMsgAttr:
			value = m.Envelope.Format()
		case FlagsMsgAttr:
			flags := make([]interface{}, len(m.Flags))
			for i, v := range m.Flags {
				flags[i] = v
			}
			value = flags
		case InternalDateMsgAttr:
			value = m.InternalDate
		case SizeMsgAttr:
			value = m.Size
		case UidMsgAttr:
			value = m.Uid
		default:
			ok = false
		}

		if ok {
			fields = append(fields, item, value)
		}
	}

	for section, literal := range m.Body {
		fields = append(fields, section.resp(), literal)
	}

	return
}

// Get the body section with the specified name. Returns nil if it's not found.
func (m *Message) GetBody(s string) *Literal {
	for section, body := range m.Body {
		if section.value == s {
			return body
		}
	}
	return nil
}

// A body section name.
// See RFC 3501 page 55.
type BodySectionName struct {
	*BodyPartName

	// If set to true, do not implicitly set the \Seen flag.
	Peek bool
	// The substring of the section requested. The first value is the position of
	// the first desired octet and the second value is the maximum number of
	// octets desired.
	Partial []int

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

	partStart := strings.Index(s, "[")
	if partStart == -1 {
		return errors.New("Invalid body section name: must contain an open bracket")
	}

	partEnd := strings.LastIndex(s, "]")
	if partEnd == -1 {
		return errors.New("Invalid body section name: must contain a close bracket")
	}

	name := s[:partStart]
	part := s[partStart+1 : partEnd]
	partial := s[partEnd+1:]

	if name == "BODY.PEEK" {
		section.Peek = true
	} else if name != "BODY" {
		return errors.New("Invalid body section name")
	}

	b := bytes.NewBufferString(part + string(cr) + string(lf))
	r := NewReader(b)
	var fields []interface{}
	if fields, err = r.ReadFields(); err != nil {
		return
	}

	section.BodyPartName = &BodyPartName{}
	if err = section.BodyPartName.parse(fields); err != nil {
		return
	}

	if len(partial) > 0 {
		if !strings.HasPrefix(partial, "<") || !strings.HasSuffix(partial, ">") {
			return errors.New("Invalid body section name: invalid partial")
		}
		partial = partial[1 : len(partial)-1]

		partialParts := strings.SplitN(partial, ".", 2)

		var from, length int
		if from, err = strconv.Atoi(partialParts[0]); err != nil {
			return errors.New("Invalid body section name: invalid partial: invalid from: " + err.Error())
		}
		section.Partial = []int{from}

		if len(partialParts) == 2 {
			if length, err = strconv.Atoi(partialParts[1]); err != nil {
				return errors.New("Invalid body section name: invalid partial: invalid length: " + err.Error())
			}
			section.Partial = append(section.Partial, length)
		}
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

	s += "[" + section.BodyPartName.String() + "]"

	if len(section.Partial) > 0 {
		s += "<"
		s += strconv.Itoa(section.Partial[0])

		if len(section.Partial) > 1 {
			s += "."
			s += strconv.Itoa(section.Partial[1])
		}

		s += ">"
	}

	return
}

func (section *BodySectionName) resp() *BodySectionName {
	var reset bool

	if section.Peek != false {
		section.Peek = false
		reset = true
	}

	if len(section.Partial) == 2 {
		section.Partial = []int{section.Partial[0]}
		reset = true
	}

	if reset && !strings.HasPrefix(section.value, "RFC822") {
		section.value = "" // Reset cached value
	}

	return section
}

// Returns a subset of the specified bytes matching the partial requested in the
// section name.
func (section *BodySectionName) ExtractPartial(b []byte) []byte {
	if len(section.Partial) != 2 {
		return b
	}

	from := section.Partial[0]
	length := section.Partial[1]
	to := from + length
	if from > len(b) {
		return nil
	}
	if to > len(b) {
		to = len(b)
	}
	return b[from:to]
}

// Parse a body section name.
func NewBodySectionName(s string) (section *BodySectionName, err error) {
	section = &BodySectionName{}
	err = section.parse(s)
	return
}

// A body part name.
type BodyPartName struct {
	// The specifier of the requested part.
	Specifier string
	// The part path. Parts indexes start at 1.
	Path []int
	// If Specifier is HEADER, contains header fields that will/won't be returned,
	// depending of the value of NotFields.
	Fields []string
	// If set to true, Fields is a blacklist of fields instead of a whitelist.
	NotFields bool
}

func (part *BodyPartName) parse(fields []interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	name, ok := fields[0].(string)
	if !ok {
		return errors.New("Invalid body section name: part name must be a string")
	}

	args := fields[1:]

	path := strings.Split(strings.ToUpper(name), ".")

	end := 0
	for i, node := range path {
		if node == "" || node == HeaderSpecifier || node == MimeSpecifier || node == TextSpecifier {
			part.Specifier = node
			end = i + 1
			break
		}

		index, err := strconv.Atoi(node)
		if err != nil {
			return errors.New("Invalid body part name: " + err.Error())
		}
		if index <= 0 {
			return errors.New("Invalid body part name: index <= 0")
		}

		part.Path = append(part.Path, index)
	}

	if part.Specifier == HeaderSpecifier && len(path) > end && path[end] == "FIELDS" && len(args) > 0 {
		end++
		if len(path) > end && path[end] == "NOT" {
			part.NotFields = true
		}

		names, ok := args[0].([]interface{})
		if !ok {
			return errors.New("Invalid body part name: HEADER.FIELDS must have a list argument")
		}

		for _, namei := range names {
			if name, ok := namei.(string); ok {
				part.Fields = append(part.Fields, name)
			}
		}
	}

	return nil
}

func (part *BodyPartName) String() (s string) {
	path := make([]string, len(part.Path))
	for i, index := range part.Path {
		path[i] = strconv.Itoa(index)
	}

	if part.Specifier != "" {
		path = append(path, part.Specifier)
	}

	if part.Specifier == HeaderSpecifier && len(part.Fields) > 0 {
		path = append(path, "FIELDS")

		if part.NotFields {
			path = append(path, "NOT")
		}
	}

	s = strings.Join(path, ".")

	if len(part.Fields) > 0 {
		s += " (" + strings.Join(part.Fields, " ") + ")"
	}

	return
}

// A message envelope, ie. message metadata from its headers.
// See RFC 3501 page 77.
type Envelope struct {
	// The message date.
	Date time.Time
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
		e.Date, _ = time.Parse(EnvelopeDateTimeFormat, date)
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
		&timeLayout{date: e.Date, layout: EnvelopeDateTimeFormat},
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
	fields := make([]interface{}, 4)

	if addr.PersonalName != "" {
		fields[0] = addr.PersonalName
	}
	if addr.AtDomainList != "" {
		fields[1] = addr.AtDomainList
	}
	if addr.MailboxName != "" {
		fields[2] = addr.MailboxName
	}
	if addr.HostName != "" {
		fields[3] = addr.HostName
	}

	return fields
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

	// True if the body structure contains extension data.
	Extended bool

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

		if i%2 == 0 {
			key = p
		} else {
			params[key] = p
			key = ""
		}
	}

	if key != "" {
		err = errors.New("Parameter list contains a key without a value")
	}
	return
}

func FormatParamList(params map[string]string) []interface{} {
	fields := []interface{}{}
	for key, value := range params {
		fields = append(fields, key, value)
	}
	return fields
}

func (bs *BodyStructure) Parse(fields []interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	// Initialize params map
	bs.Params = map[string]string{}

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
		end++

		if len(fields)-end+1 >= 4 { // Contains extension data
			bs.Extended = true

			params, _ := fields[end].([]interface{})
			bs.Params, _ = ParseParamList(params)

			bs.Disposition, _ = fields[end+1].(string)

			switch langs := fields[end+2].(type) {
			case string:
				bs.Language = []string{langs}
			case []interface{}:
				bs.Language, _ = ParseStringList(langs)
			}

			if location, ok := fields[end+3].([]interface{}); ok {
				bs.Location, _ = ParseStringList(location)
			}
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
			if len(fields)-end < 3 {
				return errors.New("Missing type-specific fields for message/rfc822")
			}

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
			if len(fields)-end < 1 {
				return errors.New("Missing type-specific fields for text/*")
			}

			bs.Lines, _ = ParseNumber(fields[end])
			end++
		}

		if len(fields)-end+1 >= 4 { // Contains extension data
			bs.Extended = true

			bs.Md5, _ = fields[end].(string)
			bs.Disposition, _ = fields[end+1].(string)

			switch langs := fields[end+2].(type) {
			case string:
				bs.Language = []string{langs}
			case []interface{}:
				bs.Language, _ = ParseStringList(langs)
			}

			if location, ok := fields[end+3].([]interface{}); ok {
				bs.Location, _ = ParseStringList(location)
			}
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

		if bs.Extended {
			extended := make([]interface{}, 4)

			if bs.Params != nil {
				extended[0] = FormatParamList(bs.Params)
			}
			if bs.Disposition != "" {
				extended[1] = bs.Disposition
			}
			if bs.Language != nil {
				extended[2] = FormatStringList(bs.Language)
			}
			if bs.Location != nil {
				extended[3] = FormatStringList(bs.Location)
			}

			fields = append(fields, extended...)
		}
	} else {
		fields = make([]interface{}, 7)
		fields[0] = bs.MimeType
		fields[1] = bs.MimeSubType
		fields[2] = FormatParamList(bs.Params)

		if bs.Id != "" {
			fields[3] = bs.Id
		}
		if bs.Description != "" {
			fields[4] = bs.Description
		}
		if bs.Encoding != "" {
			fields[5] = bs.Encoding
		}

		fields[6] = bs.Size

		// Type-specific fields
		if bs.MimeType == "message" && bs.MimeSubType == "rfc822" {
			var env interface{}
			if bs.Envelope != nil {
				env = bs.Envelope.Format()
			}

			var bsbs interface{}
			if bs.BodyStructure != nil {
				bsbs = bs.BodyStructure.Format()
			}

			fields = append(fields, env, bsbs, bs.Lines)
		}
		if bs.MimeType == "text" {
			fields = append(fields, bs.Lines)
		}

		// Extension data
		if bs.Extended {
			extended := make([]interface{}, 4)

			if bs.Md5 != "" {
				extended[0] = bs.Md5
			}
			if bs.Disposition != "" {
				extended[1] = bs.Disposition
			}
			if bs.Language != nil {
				extended[2] = FormatStringList(bs.Language)
			}
			if bs.Location != nil {
				extended[3] = FormatStringList(bs.Location)
			}

			fields = append(fields, extended...)
		}
	}

	return
}
