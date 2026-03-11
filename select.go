package imap

// SelectOptions contains options for the SELECT or EXAMINE command.
type SelectOptions struct {
	ReadOnly  bool
	CondStore bool // requires CONDSTORE
	QResync   *QResyncData
}

// QResyncData contains data for the QRESYNC SELECT/EXAMINE parameter.
type QResyncData struct {
	UIDValidity uint32
	ModSeq      uint64
	KnownUIDs   UIDSet
	SeqMatch    *QResyncSeqMatch
}

// QResyncSeqMatch contains sequence match data for the QRESYNC parameter.
type QResyncSeqMatch struct {
	SeqNums UIDSet
	UIDs    UIDSet
}

// SelectModifiedData contains data about a message modified since a given
// mod-sequence, for the QRESYNC extension.
type SelectModifiedData struct {
	SeqNum uint32
	UID    UID
	Flags  []Flag
	ModSeq uint64
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

	// QRESYNC extension
	Vanished UIDSet
	Modified []SelectModifiedData
}
