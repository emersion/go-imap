package backend

// User represents a user in the mail storage system.
type User interface {
	// Returns a list of mailboxes belonging to this user.
	ListMailboxes() ([]Mailbox, error)

	// Get a mailbox.
	GetMailbox(name string) (Mailbox, error)

	// Create a new mailbox.
	//
	// If the mailbox already exists, an error must be returned. If the mailbox
	// name is suffixed with the server's hierarchy separator character, this is a
	// declaration that the client intends to create mailbox names under this name
	// in the hierarchy.
	//
	// If the server's hierarchy separator character appears elsewhere in
	// the name, the server SHOULD create any superior hierarchical names
	// that are needed for the CREATE command to be successfully
	// completed.  In other words, an attempt to create "foo/bar/zap" on
	// a server in which "/" is the hierarchy separator character SHOULD
	// create foo/ and foo/bar/ if they do not already exist.
	//
	// If a new mailbox is created with the same name as a mailbox which
	// was deleted, its unique identifiers MUST be greater than any
	// unique identifiers used in the previous incarnation of the mailbox
	// UNLESS the new incarnation has a different unique identifier
	// validity value.
	CreateMailbox(name string) error

	// Permanently remove the mailbox with the given name. It is an error to
	// attempt to delete INBOX or a mailbox name that does not exist.
	//
	// The DELETE command MUST NOT remove inferior hierarchical names.
	// For example, if a mailbox "foo" has an inferior "foo.bar"
	// (assuming "." is the hierarchy delimiter character), removing
	// "foo" MUST NOT remove "foo.bar".
	//
	// The value of the highest-used unique identifier of the deleted
	// mailbox MUST be preserved so that a new mailbox created with the
	// same name will not reuse the identifiers of the former
	// incarnation, UNLESS the new incarnation has a different unique
	// identifier validity value.
	DeleteMailbox(name string) error
}
