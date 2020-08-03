package backend

import (
	"github.com/emersion/go-imap"
)

// Mailbox represents a mailbox belonging to a user in the mail storage system.
// A mailbox operation always deals with messages.
type Mailbox interface {
	// Name returns this mailbox name.
	Name() string

	// Closes the mailbox.
	Close() error

	// Poll requests any pending mailbox updates to be sent.
	//
	// Argument indicates whether EXPUNGE updates are permitted to be
	// sent.
	Poll(expunge bool) error

	// ListMessages returns a list of messages. seqset must be interpreted as UIDs
	// if uid is set to true and as message sequence numbers otherwise. See RFC
	// 3501 section 6.4.5 for a list of items that can be requested.
	//
	// Messages must be sent to ch. When the function returns, ch must be closed.
	ListMessages(uid bool, seqset *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error

	// SearchMessages searches messages. The returned list must contain UIDs if
	// uid is set to true, or sequence numbers otherwise.
	SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error)

	// UpdateMessagesFlags alters flags for the specified message(s).
	//
	// If the Backend implements Updater, it must notify the client immediately
	// via a message update.
	UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, operation imap.FlagsOp, silent bool, flags []string) error

	// CopyMessages copies the specified message(s) to the end of the specified
	// destination mailbox. The flags and internal date of the message(s) SHOULD
	// be preserved, and the Recent flag SHOULD be set, in the copy.
	//
	// If the destination mailbox does not exist, a server SHOULD return an error.
	// It SHOULD NOT automatically create the mailbox.
	//
	// If the Backend implements Updater, it must notify the client immediately
	// via a mailbox update.
	CopyMessages(uid bool, seqset *imap.SeqSet, dest string) error

	// Expunge permanently removes all messages that have the \Deleted flag set
	// from the currently selected mailbox.
	//
	// If the Backend implements Updater, it must notify the client immediately
	// via an expunge update.
	Expunge() error
}
