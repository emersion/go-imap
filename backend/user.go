package backend

// User represents a user in the mail storage system.
type User interface {
	// Returns a list of mailboxes belonging to this user.
	ListMailboxes() ([]Mailbox, error)
	// Get a mailbox.
	GetMailbox(name string) (Mailbox, error)
}
