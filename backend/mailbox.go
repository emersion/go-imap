package backend

import (
	"github.com/emersion/imap/common"
)

// Mailbox represents a mailbox belonging to a user in the mail storage system.
type Mailbox interface {
	// Get this mailbox info.
	Info() (*common.MailboxInfo, error)
	// Get this mailbox status.
	Status(items []string) (*common.MailboxStatus, error)
}
