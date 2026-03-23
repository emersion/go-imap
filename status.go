package imap

// StatusOptions contains options for the STATUS command.
type StatusOptions struct {
	NumMessages bool
	NumRecent   bool // Obsolete, IMAP4rev1 only. Server-only, not supported in imapclient.
	UIDNext     bool
	UIDValidity bool
	NumUnseen   bool
	NumDeleted  bool // requires IMAP4rev2 or QUOTA
	Size        bool // requires IMAP4rev2 or STATUS=SIZE

	AppendLimit    bool // requires APPENDLIMIT
	DeletedStorage bool // requires QUOTA=RES-STORAGE
	HighestModSeq  bool // requires CONDSTORE
}

// StatusData is the data returned by a STATUS command.
//
// The mailbox name is always populated. The remaining fields are optional.
type StatusData struct {
	Mailbox string

	NumMessages *uint32
	NumRecent   *uint32 // Obsolete, IMAP4rev1 only. Server-only, not supported in imapclient.
	UIDNext     UID
	UIDValidity uint32
	NumUnseen   *uint32
	NumDeleted  *uint32
	Size        *int64

	AppendLimit    *uint32
	DeletedStorage *int64
	HighestModSeq  uint64
}
