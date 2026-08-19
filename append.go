package imap

import (
	"time"
)

// AppendOptions contains options for the APPEND command.
type AppendOptions struct {
	Flags []Flag
	Time  time.Time

	// Binary sends the message as a literal8 (RFC 3516) so that arbitrary
	// binary data such as NUL bytes is preserved. The server must advertise
	// the BINARY capability or support IMAP4rev2.
	Binary bool
}

// AppendData is the data returned by an APPEND command.
type AppendData struct {
	// requires UIDPLUS or IMAP4rev2
	UID         UID
	UIDValidity uint32
}
