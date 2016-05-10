// IMAP server backend interface.
package backend

// An IMAP server backend.
// A backend operation always deals with users.
type Backend interface {
	// Authenticate a user.
	Login(username, password string) (User, error)
}
