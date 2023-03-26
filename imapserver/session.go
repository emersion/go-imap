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
	Login(username, password string) error
	Status(mailbox string, items []imap.StatusItem) (*imap.StatusData, error)
	List(w *ListWriter, ref, pattern string, options *imap.ListOptions) error
	Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error)
	Select(mailbox string, options *SelectOptions) (*imap.SelectData, error)
	Unselect() error
	Fetch(w *FetchWriter, kind NumKind, seqSet imap.SeqSet, items []imap.FetchItem) error
	Expunge(uids *imap.SeqSet) error
}
