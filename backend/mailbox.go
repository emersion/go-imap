package backend

import (
	"time"

	"github.com/emersion/go-imap/common"
)

// Mailbox represents a mailbox belonging to a user in the mail storage system.
// A mailbox operation always deals with messages.
type Mailbox interface {
	// Get this mailbox name.
	Name() string

	// Get this mailbox info.
	Info() (*common.MailboxInfo, error)

	// Get this mailbox status.
	// See RFC 3501 section 6.3.10 for a list of items that can be requested.
	Status(items []string) (*common.MailboxStatus, error)

	// Add the mailbox to the server's set of "active" or "subscribed" mailboxes.
	Subscribe() error

	// Remove the mailbox to the server's set of "active" or "subscribed"
	// mailboxes.
	Unsubscribe() error

	// Requests a checkpoint of the currently selected mailbox. A checkpoint
	// refers to any implementation-dependent housekeeping associated with the
	// mailbox (e.g., resolving the server's in-memory state of the mailbox with
	// the state on its disk). A checkpoint MAY take a non-instantaneous amount of
	// real time to complete. If a server implementation has no such housekeeping
	// considerations, CHECK is equivalent to NOOP.
	Check() error

	// Get a list of messages.
	// seqset must be interpreted as UIDs if uid is set to true and as message
	// sequence numbers otherwise.
	// See RFC 3501 section 6.4.5 for a list of items that can be requested.
	ListMessages(uid bool, seqset *common.SeqSet, items []string) ([]*common.Message, error)

	// Search messages.
	SearchMessages(uid bool, criteria *common.SearchCriteria) ([]uint32, error)

	// Append a new message to this mailbox. The \Recent flag will be added no
	// matter flags is empty or not. If date is nil, the current time will be
	// used.
	CreateMessage(flags []string, date *time.Time, body []byte) error

	// Alter flags for the specified message(s).
	UpdateMessagesFlags(uid bool, seqset *common.SeqSet, operation common.FlagsOp, flags []string) error

	// Copy the specified message(s) to the end of the specified destination
	// mailbox. The flags and internal date of the message(s) SHOULD be preserved,
	// and the Recent flag SHOULD be set, in the copy.
	//
	// If the destination mailbox does not exist, a server SHOULD return an error.
	// It SHOULD NOT automatically create the mailbox.
	CopyMessages(uid bool, seqset *common.SeqSet, dest string) error

	// Permanently removes all messages that have the \Deleted flag set from the
	// currently selected mailbox.
	Expunge() error
}
