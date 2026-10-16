package imap

// SelectOptions contains options for the SELECT or EXAMINE command.
type SelectOptions struct {
	ReadOnly  bool
	CondStore bool             // requires CONDSTORE
	QResync   *QResyncOptions // requires QRESYNC
}

// QResyncOptions is the payload of the (QRESYNC ...) select modifier
// (RFC 7162 §3.2). The client tells the server "I last knew this
// mailbox at UIDValidity = U, HIGHESTMODSEQ = M; tell me what has
// changed since". KnownUIDs (optional) is the client's UID set —
// servers MAY restrict their VANISHED report to that subset. The
// per-cache reconciliation pair (KnownSeqSet, KnownUIDSet) is RFC
// 7162 §3.2.5; this implementation surfaces them on the struct but
// does not require the server to honour them.
type QResyncOptions struct {
	UIDValidity uint32
	ModSeq      uint64
	KnownUIDs   UIDSet
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

	// Vanished, when non-empty, is reported as
	// "* VANISHED (EARLIER) <uids>" before the tagged OK. The
	// server populates it during a QRESYNC SELECT with the UIDs
	// that have been expunged since the client's last known
	// HIGHESTMODSEQ. RFC 7162 §3.2.
	Vanished UIDSet
}
