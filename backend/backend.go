// IMAP server backend interface.
package backend

// An IMAP server backend.
type Backend interface {
	// Authenticate a user.
	Login(username, password string) (User, error)
}
