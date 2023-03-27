package imapserver

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

var errAuthFailed = &imap.Error{
	Type: imap.StatusResponseTypeNo,
	Code: imap.ResponseCodeAuthenticationFailed,
	Text: "Authentication failed",
}

// ErrAuthFailed is returned by Session.Login on authentication failure.
var ErrAuthFailed = errAuthFailed

// NumKind describes how a number should be interpreted: either as a sequence
// number, either as a UID.
type NumKind int

const (
	NumKindSeq NumKind = 1 + iota
	NumKindUID
)

// String implements fmt.Stringer.
func (kind NumKind) String() string {
	switch kind {
	case NumKindSeq:
		return "seq"
	case NumKindUID:
		return "uid"
	default:
		panic(fmt.Errorf("imapserver: unknown NumKind %d", kind))
	}
}

// Session is an IMAP session.
type Session interface {
	Close() error

	// Not authenticated state
	Login(username, password string) error

	// Authenticated state
	Select(mailbox string, options *SelectOptions) (*imap.SelectData, error)
	Create(mailbox string) error
	Delete(mailbox string) error
	Rename(mailbox, newName string) error
	Subscribe(mailbox string) error
	Unsubscribe(mailbox string) error
	List(w *ListWriter, ref, pattern string, options *imap.ListOptions) error
	Namespace() (*imap.NamespaceData, error)
	Status(mailbox string, items []imap.StatusItem) (*imap.StatusData, error)
	Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error)

	// Selected state
	Unselect() error
	Expunge(uids *imap.SeqSet) error
	Search(kind NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error)
	Fetch(w *FetchWriter, kind NumKind, seqSet imap.SeqSet, items []imap.FetchItem) error
	Store(w *FetchWriter, kind NumKind, seqSet imap.SeqSet, flags *imap.StoreFlags) error
	Copy(kind NumKind, seqSet imap.SeqSet, dest string) error
	Move(kind NumKind, seqSet imap.SeqSet, dest string) error
}
