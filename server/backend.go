package server

// An IMAP server backend.
type Backend interface {
	// Authenticate a user.
	Login(username, password string) error
}
