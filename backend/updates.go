package backend

import (
	"github.com/emersion/go-imap"
)

type Update interface {
	Update()
}

type Conn interface {
	// SendUpdate sends unilateral update to the connection.
	//
	// Backend should not call this method when no backend method is running
	// as Conn is not guaranteed to be in a consistent state otherwise.
	SendUpdate(upd Update) error
}

// StatusUpdate is a status update. See RFC 3501 section 7.1 for a list of
// status responses.
type StatusUpdate struct {
	*imap.StatusResp
}

func (*StatusUpdate) Update() {}

// MailboxUpdate is a mailbox update.
type MailboxUpdate struct {
	*imap.MailboxStatus
}

func (*MailboxUpdate) Update() {}

// MailboxInfoUpdate is a maiblox info update.
type MailboxInfoUpdate struct {
	*imap.MailboxInfo
}

func (*MailboxInfoUpdate) Update() {}

// MessageUpdate is a message update.
type MessageUpdate struct {
	*imap.Message
}

func (*MessageUpdate) Update() {}

// ExpungeUpdate is an expunge update.
type ExpungeUpdate struct {
	SeqNum uint32
}

func (*ExpungeUpdate) Update() {}
