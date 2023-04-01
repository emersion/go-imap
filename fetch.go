package imap

import (
	"fmt"
	"strings"
)

// FetchItem is a message data item which can be requested by a FETCH command.
type FetchItem interface {
	fetchItem()
}

var (
	_ FetchItem = FetchItemKeyword("")
	_ FetchItem = (*FetchItemBodySection)(nil)
	_ FetchItem = (*FetchItemBinarySection)(nil)
	_ FetchItem = (*FetchItemBinarySectionSize)(nil)
)

// FetchItemKeyword is a FETCH item described by a single keyword.
type FetchItemKeyword string

func (FetchItemKeyword) fetchItem() {}

var (
	// Macros
	FetchItemAll  FetchItem = FetchItemKeyword("ALL")
	FetchItemFast FetchItem = FetchItemKeyword("FAST")
	FetchItemFull FetchItem = FetchItemKeyword("FULL")

	FetchItemBody          FetchItem = FetchItemKeyword("BODY")
	FetchItemBodyStructure FetchItem = FetchItemKeyword("BODYSTRUCTURE")
	FetchItemEnvelope      FetchItem = FetchItemKeyword("ENVELOPE")
	FetchItemFlags         FetchItem = FetchItemKeyword("FLAGS")
	FetchItemInternalDate  FetchItem = FetchItemKeyword("INTERNALDATE")
	FetchItemRFC822Size    FetchItem = FetchItemKeyword("RFC822.SIZE")
	FetchItemUID           FetchItem = FetchItemKeyword("UID")
)

type PartSpecifier string

const (
	PartSpecifierNone   PartSpecifier = ""
	PartSpecifierHeader PartSpecifier = "HEADER"
	PartSpecifierMIME   PartSpecifier = "MIME"
	PartSpecifierText   PartSpecifier = "TEXT"
)

type SectionPartial struct {
	Offset, Size int64
}

// FetchItemBodySection is a FETCH BODY[] data item.
type FetchItemBodySection struct {
	Specifier       PartSpecifier
	Part            []int
	HeaderFields    []string
	HeaderFieldsNot []string
	Partial         *SectionPartial
	Peek            bool
}

func (*FetchItemBodySection) fetchItem() {}

// FetchItemBinarySection is a FETCH BINARY[] data item.
type FetchItemBinarySection struct {
	Part    []int
	Partial *SectionPartial
	Peek    bool
}

func (*FetchItemBinarySection) fetchItem() {}

// FetchItemBinarySectionSize is a FETCH BINARY.SIZE[] data item.
type FetchItemBinarySectionSize struct {
	Part []int
}

func (*FetchItemBinarySectionSize) fetchItem() {}

// Envelope is the envelope structure of a message.
type Envelope struct {
	Date      string // see net/mail.ParseDate
	Subject   string
	From      []Address
	Sender    []Address
	ReplyTo   []Address
	To        []Address
	Cc        []Address
	Bcc       []Address
	InReplyTo string
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

type BodyStructureMessageRFC822 struct {
	Envelope      *Envelope
	BodyStructure BodyStructure
	NumLines      int64
}

type BodyStructureText struct {
	NumLines int64
}

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

type BodyStructureMultiPartExt struct {
	Params      map[string]string
	Disposition *BodyStructureDisposition
	Language    []string
	Location    string
}

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
