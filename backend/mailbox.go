package backend

import (
	"github.com/emersion/imap/common"
)

// Mailbox represents a mailbox belonging to a user in the mail storage system.
type Mailbox interface {
	// Get this mailbox info.
	Info() (*common.MailboxInfo, error)
	// Get this mailbox status.
	// See RFC 3501 section 6.3.10 for a list of items that can be requested.
	Status(items []string) (*common.MailboxStatus, error)

	// Get a list of messages.
	// seqset must be interpreted as UIDs if uid is set to true and as message
	// sequence numbers otherwise.
	// See RFC 3501 section 6.4.5 for a list of items that can be requested.
	ListMessages(uid bool, seqset *common.SeqSet, items []string) ([]*common.Message, error)

	// Search messages.
	SearchMessages(uid bool, criteria *common.SearchCriteria) ([]uint32, error)
}
