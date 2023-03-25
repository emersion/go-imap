package imapserver

import (
	"github.com/emersion/go-imap/v2"
)

var errAuthFailed = &imap.Error{
	Type: imap.StatusResponseTypeNo,
	Code: imap.ResponseCodeAuthenticationFailed,
	Text: "Authentication failed",
}

// ErrAuthFailed is returned by Session.Login on authentication failure.
var ErrAuthFailed = errAuthFailed

// Session is an IMAP session.
type Session interface {
	Close() error
	Login(username, password string) error
	Status(mailbox string, items []imap.StatusItem) (*imap.StatusData, error)
	List(ref, pattern string, options *imap.ListOptions) ([]imap.ListData, error)
	Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error)
	Select(mailbox string, options *SelectOptions) (*imap.SelectData, error)
	Unselect() error
	Fetch(w *FetchWriter, seqSet imap.SeqSet, items []imap.FetchItem) error
}
