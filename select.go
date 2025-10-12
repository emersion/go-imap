package imap

// SelectOptions contains options for the SELECT or EXAMINE command.
type SelectOptions struct {
	ReadOnly  bool
	CondStore bool // requires CONDSTORE

	// QRESYNC parameters (requires QRESYNC extension, RFC 5162)
	QResync *SelectQResyncOptions
}

// SelectQResyncOptions contains QRESYNC parameters for SELECT.
type SelectQResyncOptions struct {
	UIDValidity uint32
	ModSeq      uint64
	KnownUIDs   *UIDSet // optional
	SeqMatchData *SelectSeqMatchData // optional
}

// SelectSeqMatchData contains sequence match data for QRESYNC.
type SelectSeqMatchData struct {
	KnownSeqSet SeqSet
	KnownUIDSet UIDSet
}

// SelectData is the data returned by a SELECT command.
//
// In the old RFC 2060, PermanentFlags, UIDNext and UIDValidity are optional.
type SelectData struct {
	// Flags defined for this mailbox
	Flags []Flag
	// Flags that the client can change permanently
	PermanentFlags []Flag
	// Number of messages in this mailbox (aka. "EXISTS")
	NumMessages uint32
	// Sequence number of the first unseen message. Obsolete, IMAP4rev1 only.
	// Server-only, not supported in imapclient.
	FirstUnseenSeqNum uint32
	// Number of recent messages in this mailbox. Obsolete, IMAP4rev1 only.
	// Server-only, not supported in imapclient.
	NumRecent   uint32
	UIDNext     UID
	UIDValidity uint32

	List *ListData // requires IMAP4rev2

	HighestModSeq uint64 // requires CONDSTORE

	// UIDs of messages that were expunged.
	// Requires QRESYNC extension (RFC 4551/7162).
	VanishedUIDs UIDSet // requires QRESYNC
}
