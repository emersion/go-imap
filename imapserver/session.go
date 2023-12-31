package imapserver

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
	"github.com/emersion/go-sasl"
)

var errAuthFailed = &imap.Error{
	Type: imap.StatusResponseTypeNo,
	Code: imap.ResponseCodeAuthenticationFailed,
	Text: "Authentication failed",
}

// ErrAuthFailed is returned by Session.Login on authentication failure.
var ErrAuthFailed = errAuthFailed

// GreetingData is the data associated with an IMAP greeting.
type GreetingData struct {
	PreAuth bool
}

// NumKind describes how a number should be interpreted: either as a sequence
// number, either as a UID.
type NumKind int

const (
	NumKindSeq = NumKind(imapwire.NumKindSeq)
	NumKindUID = NumKind(imapwire.NumKindUID)
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

func (kind NumKind) wire() imapwire.NumKind {
	return imapwire.NumKind(kind)
}

// Session is an IMAP session.
type Session interface {
	Close() error

	// Not authenticated state
	Login(username, password string) error

	// Authenticated state
	Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error)
	Create(mailbox string, options *imap.CreateOptions) error
	Delete(mailbox string) error
	Rename(mailbox, newName string) error
	Subscribe(mailbox string) error
	Unsubscribe(mailbox string) error
	List(w *ListWriter, ref string, patterns []string, options *imap.ListOptions) error
	Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error)
	Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error)
	Poll(w *UpdateWriter, allowExpunge bool) error
	Idle(w *UpdateWriter, stop <-chan struct{}) error

	// Selected state
	Unselect() error
	Expunge(w *ExpungeWriter, uids *imap.UIDSet) error
	Search(kind NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error)
	Fetch(w *FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error
	Store(w *FetchWriter, numSet imap.NumSet, flags *imap.StoreFlags, options *imap.StoreOptions) error
	Copy(numSet imap.NumSet, dest string) (*imap.CopyData, error)
}

// SessionNamespace is an IMAP session which supports NAMESPACE.
type SessionNamespace interface {
	Session

	// Authenticated state
	Namespace() (*imap.NamespaceData, error)
}

// SessionMove is an IMAP session which supports MOVE.
type SessionMove interface {
	Session

	// Selected state
	Move(w *MoveWriter, numSet imap.NumSet, dest string) error
}

// SessionIMAP4rev2 is an IMAP session which supports IMAP4rev2.
type SessionIMAP4rev2 interface {
	Session
	SessionNamespace
	SessionMove
}

// SessionSASL is an IMAP session which supports its own set of SASL
// authentication mechanisms.
type SessionSASL interface {
	Session
	AuthenticateMechanisms() []string
	Authenticate(mech string) (sasl.Server, error)
}

// SessionUnauthenticate is an IMAP session which supports UNAUTHENTICATE.
type SessionUnauthenticate interface {
	Session

	// Authenticated state
	Unauthenticate() error
}
