package imap

import (
	"fmt"
	"strings"
	"time"
)

// FetchOptions contains options for the FETCH command.
type FetchOptions struct {
	// Fields to fetch
	BodyStructure     *FetchItemBodyStructure
	Envelope          bool
	Flags             bool
	InternalDate      bool
	RFC822Size        bool
	UID               bool
	BodySection       []*FetchItemBodySection
	BinarySection     []*FetchItemBinarySection     // requires IMAP4rev2 or BINARY
	BinarySectionSize []*FetchItemBinarySectionSize // requires IMAP4rev2 or BINARY
	ModSeq            bool                          // requires CONDSTORE

	ChangedSince uint64 // requires CONDSTORE

	// CustomAttributes are server-defined attribute names (e.g. "X-GM-MSGID")
	// requested verbatim. Responses are parsed via the client's
	// imapclient.Options.CustomAttributeDecoders registry.
	CustomAttributes []string
}

// Clone returns a deep copy of o. The returned value shares no mutable state
// with the original: pointer fields are followed and re-allocated, slices are
// copied. A nil receiver yields a nil result.
func (o *FetchOptions) Clone() *FetchOptions {
	if o == nil {
		return nil
	}
	c := *o
	if o.BodyStructure != nil {
		bs := *o.BodyStructure
		c.BodyStructure = &bs
	}
	if o.BodySection != nil {
		c.BodySection = make([]*FetchItemBodySection, len(o.BodySection))
		for i, s := range o.BodySection {
			c.BodySection[i] = s.Clone()
		}
	}
	if o.BinarySection != nil {
		c.BinarySection = make([]*FetchItemBinarySection, len(o.BinarySection))
		for i, s := range o.BinarySection {
			c.BinarySection[i] = s.Clone()
		}
	}
	if o.BinarySectionSize != nil {
		c.BinarySectionSize = make([]*FetchItemBinarySectionSize, len(o.BinarySectionSize))
		for i, s := range o.BinarySectionSize {
			c.BinarySectionSize[i] = s.Clone()
		}
	}
	if o.CustomAttributes != nil {
		c.CustomAttributes = append([]string(nil), o.CustomAttributes...)
	}
	return &c
}

// WithAttributes returns a deep copy of o with the given custom attribute
// names appended to CustomAttributes. The original is left untouched, so the
// method is safe to chain off a shared base FetchOptions. A nil receiver
// produces a fresh FetchOptions carrying just attrs.
func (o *FetchOptions) WithAttributes(attrs ...string) *FetchOptions {
	c := o.Clone()
	if c == nil {
		c = &FetchOptions{}
	}
	if len(attrs) > 0 {
		c.CustomAttributes = append(c.CustomAttributes, attrs...)
	}
	return c
}

// FetchItemBodyStructure contains FETCH options for the body structure.
type FetchItemBodyStructure struct {
	Extended bool
}

// PartSpecifier describes whether to fetch a part's header, body, or both.
type PartSpecifier string

const (
	PartSpecifierNone   PartSpecifier = ""
	PartSpecifierHeader PartSpecifier = "HEADER"
	PartSpecifierMIME   PartSpecifier = "MIME"
	PartSpecifierText   PartSpecifier = "TEXT"
)

// SectionPartial describes a byte range when fetching a message's payload.
type SectionPartial struct {
	Offset, Size int64
}

// FetchItemBodySection is a FETCH BODY[] data item.
//
// To fetch the whole body of a message, use the zero FetchItemBodySection:
//
//	imap.FetchItemBodySection{}
//
// To fetch only a specific part, use the Part field:
//
//	imap.FetchItemBodySection{Part: []int{1, 2, 3}}
//
// To fetch only the header of the message, use the Specifier field:
//
//	imap.FetchItemBodySection{Specifier: imap.PartSpecifierHeader}
type FetchItemBodySection struct {
	Specifier       PartSpecifier
	Part            []int
	HeaderFields    []string
	HeaderFieldsNot []string
	Partial         *SectionPartial
	Peek            bool
}

// Clone returns a deep copy of s. Returns nil for a nil receiver.
func (s *FetchItemBodySection) Clone() *FetchItemBodySection {
	if s == nil {
		return nil
	}
	c := *s
	if s.Part != nil {
		c.Part = append([]int(nil), s.Part...)
	}
	if s.HeaderFields != nil {
		c.HeaderFields = append([]string(nil), s.HeaderFields...)
	}
	if s.HeaderFieldsNot != nil {
		c.HeaderFieldsNot = append([]string(nil), s.HeaderFieldsNot...)
	}
	if s.Partial != nil {
		p := *s.Partial
		c.Partial = &p
	}
	return &c
}

// FetchItemBinarySection is a FETCH BINARY[] data item.
type FetchItemBinarySection struct {
	Part    []int
	Partial *SectionPartial
	Peek    bool
}

// Clone returns a deep copy of s. Returns nil for a nil receiver.
func (s *FetchItemBinarySection) Clone() *FetchItemBinarySection {
	if s == nil {
		return nil
	}
	c := *s
	if s.Part != nil {
		c.Part = append([]int(nil), s.Part...)
	}
	if s.Partial != nil {
		p := *s.Partial
		c.Partial = &p
	}
	return &c
}

// FetchItemBinarySectionSize is a FETCH BINARY.SIZE[] data item.
type FetchItemBinarySectionSize struct {
	Part []int
}

// Clone returns a deep copy of s. Returns nil for a nil receiver.
func (s *FetchItemBinarySectionSize) Clone() *FetchItemBinarySectionSize {
	if s == nil {
		return nil
	}
	c := *s
	if s.Part != nil {
		c.Part = append([]int(nil), s.Part...)
	}
	return &c
}

// Envelope is the envelope structure of a message.
//
// The subject and addresses are UTF-8 (ie, not in their encoded form). The
// In-Reply-To and Message-ID values contain message identifiers without angle
// brackets.
type Envelope struct {
	Date      time.Time
	Subject   string
	From      []Address
	Sender    []Address
	ReplyTo   []Address
	To        []Address
	Cc        []Address
	Bcc       []Address
	InReplyTo []string
	MessageID string
}

// Address represents a sender or recipient of a message.
type Address struct {
	Name    string
	Mailbox string
	Host    string
}

// Addr returns the e-mail address in the form "foo@example.org".
//
// If the address is a start or end of group, the empty string is returned.
func (addr *Address) Addr() string {
	if addr.Mailbox == "" || addr.Host == "" {
		return ""
	}
	return addr.Mailbox + "@" + addr.Host
}

// IsGroupStart returns true if this address is a start of group marker.
//
// In that case, Mailbox contains the group name phrase.
func (addr *Address) IsGroupStart() bool {
	return addr.Host == "" && addr.Mailbox != ""
}

// IsGroupEnd returns true if this address is a end of group marker.
func (addr *Address) IsGroupEnd() bool {
	return addr.Host == "" && addr.Mailbox == ""
}

// BodyStructure describes the body structure of a message.
//
// A BodyStructure value is either a *BodyStructureSinglePart or a
// *BodyStructureMultiPart.
type BodyStructure interface {
	// MediaType returns the MIME type of this body structure, e.g. "text/plain".
	MediaType() string
	// Walk walks the body structure tree, calling f for each part in the tree,
	// including bs itself. The parts are visited in DFS pre-order.
	Walk(f BodyStructureWalkFunc)
	// Disposition returns the body structure disposition, if available.
	Disposition() *BodyStructureDisposition

	bodyStructure()
}

var (
	_ BodyStructure = (*BodyStructureSinglePart)(nil)
	_ BodyStructure = (*BodyStructureMultiPart)(nil)
)

// BodyStructureSinglePart is a body structure with a single part.
type BodyStructureSinglePart struct {
	Type, Subtype string
	Params        map[string]string
	ID            string
	Description   string
	Encoding      string
	Size          uint32

	MessageRFC822 *BodyStructureMessageRFC822 // only for "message/rfc822"
	Text          *BodyStructureText          // only for "text/*"
	Extended      *BodyStructureSinglePartExt
}

func (bs *BodyStructureSinglePart) MediaType() string {
	return strings.ToLower(bs.Type) + "/" + strings.ToLower(bs.Subtype)
}

func (bs *BodyStructureSinglePart) Walk(f BodyStructureWalkFunc) {
	f([]int{1}, bs)
}

func (bs *BodyStructureSinglePart) Disposition() *BodyStructureDisposition {
	if bs.Extended == nil {
		return nil
	}
	return bs.Extended.Disposition
}

// Filename decodes the body structure's filename, if any.
func (bs *BodyStructureSinglePart) Filename() string {
	var filename string
	if bs.Extended != nil && bs.Extended.Disposition != nil {
		filename = bs.Extended.Disposition.Params["filename"]
	}
	if filename == "" {
		// Note: using "name" in Content-Type is discouraged
		filename = bs.Params["name"]
	}
	return filename
}

func (*BodyStructureSinglePart) bodyStructure() {}

// BodyStructureMessageRFC822 contains metadata specific to RFC 822 parts for
// BodyStructureSinglePart.
type BodyStructureMessageRFC822 struct {
	Envelope      *Envelope
	BodyStructure BodyStructure
	NumLines      int64
}

// BodyStructureText contains metadata specific to text parts for
// BodyStructureSinglePart.
type BodyStructureText struct {
	NumLines int64
}

// BodyStructureSinglePartExt contains extended body structure data for
// BodyStructureSinglePart.
type BodyStructureSinglePartExt struct {
	Disposition *BodyStructureDisposition
	Language    []string
	Location    string
}

// BodyStructureMultiPart is a body structure with multiple parts.
type BodyStructureMultiPart struct {
	Children []BodyStructure
	Subtype  string

	Extended *BodyStructureMultiPartExt
}

func (bs *BodyStructureMultiPart) MediaType() string {
	return "multipart/" + strings.ToLower(bs.Subtype)
}

func (bs *BodyStructureMultiPart) Walk(f BodyStructureWalkFunc) {
	bs.walk(f, nil)
}

func (bs *BodyStructureMultiPart) walk(f BodyStructureWalkFunc, path []int) {
	if !f(path, bs) {
		return
	}

	pathBuf := make([]int, len(path))
	copy(pathBuf, path)
	for i, part := range bs.Children {
		num := i + 1
		partPath := append(pathBuf, num)

		switch part := part.(type) {
		case *BodyStructureSinglePart:
			f(partPath, part)
		case *BodyStructureMultiPart:
			part.walk(f, partPath)
		default:
			panic(fmt.Errorf("unsupported body structure type %T", part))
		}
	}
}

func (bs *BodyStructureMultiPart) Disposition() *BodyStructureDisposition {
	if bs.Extended == nil {
		return nil
	}
	return bs.Extended.Disposition
}

func (*BodyStructureMultiPart) bodyStructure() {}

// BodyStructureMultiPartExt contains extended body structure data for
// BodyStructureMultiPart.
type BodyStructureMultiPartExt struct {
	Params      map[string]string
	Disposition *BodyStructureDisposition
	Language    []string
	Location    string
}

// BodyStructureDisposition describes the content disposition of a part
// (specified in the Content-Disposition header field).
type BodyStructureDisposition struct {
	Value  string
	Params map[string]string
}

// BodyStructureWalkFunc is a function called for each body structure visited
// by BodyStructure.Walk.
//
// The path argument contains the IMAP part path.
//
// The function should return true to visit all of the part's children or false
// to skip them.
type BodyStructureWalkFunc func(path []int, part BodyStructure) (walkChildren bool)
