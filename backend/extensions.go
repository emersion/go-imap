package backend

import (
	"github.com/emersion/go-imap"
)

// ExpungeSeqSet may be passed to Mailbox.Expunge() to restrict message
// deletion to the specified UIDs sequence set.
//
// See RFC 2359 for details.
type ExpungeSeqSet struct {
	*imap.SeqSet
}

func (ExpungeSeqSet) ExtOption() {}

// CopyUIDs must be returned as a result for CopyMessages by backends that
// implement UIDPLUS extension.
//
// See RFC 2359 for value details.
type CopyUIDs struct {
	Source      *imap.SeqSet
	UIDValidity uint32
	Dest        *imap.SeqSet
}

func (CopyUIDs) ExtResult() {}

// AppendUID must be returned as a result for CreateMessage by backend that
// implement UIDPLUS extension.
//
// See RFC 2359 for value details.
type AppendUID struct {
	UIDValidity uint32
	UID         uint32
}

func (AppendUID) ExtResult() {}
