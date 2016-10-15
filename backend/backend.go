// IMAP server backend interface.
package backend

import "errors"

// Error returned by Backend.Login when a username or a password is incorrect.
var ErrInvalidCredentials = errors.New("Invalid credentials")

// An IMAP server backend.
// A backend operation always deals with users.
type Backend interface {
	// Login authenticate a user. If the username or the password is incorrect,
	// return ErrInvalidCredentials.
	Login(username, password string) (User, error)
}
