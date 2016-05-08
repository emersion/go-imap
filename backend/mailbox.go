package backend

import (
	"github.com/emersion/imap/common"
)

// Mailbox represents a mailbox belonging to a user in the mail storage system.
type Mailbox interface {
	Info() (*common.MailboxInfo, error)
}
