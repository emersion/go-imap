package backend

import (
	"github.com/emersion/go-imap"
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

	// A channel that will be closed once the update has been processed.
	done chan struct{}
}

// Done returns a channel that is closed when the update has been broadcast to
// all clients.
func (u *Update) Done() <-chan struct{} {
	if u.done == nil {
		u.done = make(chan struct{})
	}
	return u.done
}

// DoneUpdate marks an update as done.
// TODO: remove this function
func DoneUpdate(u *Update) {
	if u.done != nil {
		close(u.done)
	}
}

// StatusUpdate is a status update. See RFC 3501 section 7.1 for a list of
// status responses.
type StatusUpdate struct {
	Update
	*imap.StatusResp
}

// MailboxUpdate is a mailbox update.
type MailboxUpdate struct {
	Update
	*imap.MailboxStatus
}

// MessageUpdate is a message update.
type MessageUpdate struct {
	Update
	*imap.Message
}

// ExpungeUpdate is an expunge update.
type ExpungeUpdate struct {
	Update
	SeqNum uint32
}

// Updater is a Backend that implements Updater is able to send unilateral
// backend updates. Backends not implementing this interface don't correctly
// send unilateral updates, for instance if a user logs in from two connections
// and deletes a message from one of them, the over is not aware that such a
// mesage has been deleted. More importantly, backends implementing Updater can
// notify the user for external updates such as new message notifications.
type Updater interface {
	// Updates returns a set of channels where updates are sent to.
	Updates() <-chan interface{}
}

// UpdaterMailbox is a Mailbox that implements UpdaterMailbox is able to poll
// updates for new messages or message status updates during a period of
// inactivity.
type UpdaterMailbox interface {
	// Poll requests mailbox updates.
	Poll() error
}

// WaitUpdates returns a channel that's closed when all provided updates have
// been dispatched to all clients. It panics if one of the provided value is
// not an update.
func WaitUpdates(updates ...interface{}) <-chan struct{} {
	done := make(chan struct{})

	var chs []<-chan struct{}
	for _, u := range updates {
		uu, ok := u.(interface {
			Done() <-chan struct{}
		})
		if !ok {
			panic("imap: cannot wait for update: provided value is not a valid update")
		}

		chs = append(chs, uu.Done())
	}

	go func() {
		// Wait for all updates to be sent
		for _, ch := range chs {
			<-ch
		}
		close(done)
	}()

	return done
}
