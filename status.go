package imap

// StatusItem is a data item which can be requested by a STATUS command.
type StatusItem string

const (
	StatusItemNumMessages StatusItem = "MESSAGES"
	StatusItemUIDNext     StatusItem = "UIDNEXT"
	StatusItemUIDValidity StatusItem = "UIDVALIDITY"
	StatusItemNumUnseen   StatusItem = "UNSEEN"
	StatusItemNumDeleted  StatusItem = "DELETED" // requires IMAP4rev2 or QUOTA
	StatusItemSize        StatusItem = "SIZE"    // requires IMAP4rev2 or STATUS=SIZE

	StatusItemAppendLimit    StatusItem = "APPENDLIMIT"     // requires APPENDLIMIT
	StatusItemDeletedStorage StatusItem = "DELETED-STORAGE" // requires QUOTA=RES-STORAGE
)

// StatusData is the data returned by a STATUS command.
//
// The mailbox name is always populated. The remaining fields are optional.
type StatusData struct {
	Mailbox string

	NumMessages *uint32
	UIDNext     uint32
	UIDValidity uint32
	NumUnseen   *uint32
	NumDeleted  *uint32
	Size        *int64

	AppendLimit    *uint32
	DeletedStorage *int64
}
