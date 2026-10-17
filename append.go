package imap

import (
	"time"
)

// AppendOptions contains options for the APPEND command.
type AppendOptions struct {
	Flags []Flag
	Time  time.Time
}

// AppendData is the data returned by an APPEND command.
type AppendData struct {
	// requires UIDPLUS or IMAP4rev2
	UID         UID
	UIDValidity uint32
}

// MultiAppendData is the data returned by an APPEND command with multiple messages.
type MultiAppendData struct {
	// requires UIDPLUS or IMAP4rev2
	UID         UID
	UIDs        UIDSet
	UIDValidity uint32
}
