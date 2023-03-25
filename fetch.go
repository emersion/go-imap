package imap

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
