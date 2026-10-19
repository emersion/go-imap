package imap

// SelectOptions contains options for the SELECT or EXAMINE command.
type SelectOptions struct {
	ReadOnly  bool
	CondStore bool // requires CONDSTORE

	// QResync requests a QRESYNC mailbox resynchronization (RFC 7162). It
	// requires QRESYNC to be enabled. QRESYNC implies CONDSTORE, so CondStore
	// is ignored when QResync is set.
	QResync *QResyncOptions
}

// QResyncOptions contains QRESYNC parameters for the SELECT or EXAMINE command
// (RFC 7162). The server replies with VANISHED (EARLIER) and FETCH responses to
// resynchronize the client's view of the mailbox.
type QResyncOptions struct {
	UIDValidity uint32
	ModSeq      uint64

	// KnownUIDs optionally restricts the set of UIDs the server reports on.
	KnownUIDs UIDSet
	// SeqMatchData optionally provides the client's known sequence-number to
	// UID mapping, letting the server detect expunges it would otherwise miss.
	// It requires KnownUIDs to be set.
	SeqMatchData *SeqMatchData
}

// SeqMatchData is the client's known sequence-number-to-UID mapping sent as part
// of a QRESYNC SELECT or EXAMINE command (RFC 7162).
type SeqMatchData struct {
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

	// VanishedUIDs are the UIDs reported as expunged by a VANISHED (EARLIER)
	// response during a QRESYNC SELECT or EXAMINE. Requires QRESYNC.
	VanishedUIDs UIDSet
}
