package backend

import (
	"github.com/emersion/go-imap/common"
)

// Update contains user and mailbox information about an unilateral backend
// update.
type Update struct {
	// The user targeted by this update. If empty, all connected users will
	// be notified.
	Username string
	// The mailbox targeted by this update. If empty, the update targets all
	// mailboxes.
	Mailbox string
}

// A status update. See RFC 3501 section 7.1 for a list of status responses.
type StatusUpdate struct {
	Update
	*common.StatusResp
}

// A mailbox update.
type MailboxUpdate struct {
	Update
	*common.MailboxStatus
}

// A message update.
type MessageUpdate struct {
	Update
	*common.Message
}

// An expunge update.
type ExpungeUpdate struct {
	Update
	SeqNum uint32
}

// Updates contains channels where unilateral backend updates will be sent.
type Updates struct {
	Statuses chan *StatusUpdate
	Mailboxes chan *MailboxUpdate
	Messages chan *MessageUpdate
	Expunges chan *ExpungeUpdate
}

// NewUpdates initializes a new Updates struct.
func NewUpdates() (up *Updates) {
	return &Updates{
		Statuses: make(chan *StatusUpdate),
		Mailboxes: make(chan *MailboxUpdate),
		Messages: make(chan *MessageUpdate),
		Expunges: make(chan *ExpungeUpdate),
	}
}

// A Backend that implements Updater is able to send unilateral backend updates.
// Backends not implementing this interface don't correctly send unilateral
// updates, for instance if a user logs in from two connections and deletes a
// message from one of them, the over is not aware that such a mesage has been
// deleted. More importantly, backends implementing Updater can notify the user
// for external updates such as new message notifications.
type Updater interface {
	// Updates returns a set of channels where updates are sent to.
	Updates() *Updates
}

// A Mailbox that implements UpdaterMailbox is able to poll updates for new
// messages or message status updates during a period of inactivity.
type UpdaterMailbox interface {
	// Poll requests mailbox updates.
	Poll() error
}
