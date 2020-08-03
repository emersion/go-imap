// Package backend defines an IMAP server backend interface.
package backend

import (
	"errors"

	"github.com/emersion/go-imap"
)

// ErrInvalidCredentials is returned by Backend.Login when a username or a
// password is incorrect.
var ErrInvalidCredentials = errors.New("Invalid credentials")

type Extension string

// Backend is an IMAP server backend. A backend operation always deals with
// users.
type Backend interface {
	// Login authenticates a user. If the username or the password is incorrect,
	// it returns ErrInvalidCredentials.
	Login(connInfo *imap.ConnInfo, username, password string) (User, error)

	// SupportedExtensions returns the list of extension identifiers that
	// are understood by the backend. Note that values are not capability names
	// even though some may match corresponding names. In particular,
	// parameters for capability names are not included.
	//
	// Result should contain an entry for each extension that is implemented
	// by go-imap directly. There is no requirement to include extensions
	// that are provided by separate libraries (e.g. go-imap-id), though
	// it would not hurt - unknown values are silently ignored.
	SupportedExtensions() []Extension
}

// ExtensionOption is an optional argument defined by IMAP extension
// that may be passed to the backend.
//
// Backend implementation is supposed to use type assertions to determine
// actual option data and the action needed.
// Backend implementation SHOULD fail the command if it seen
// an unknown ExtensionOption type passed to it.
type ExtensionOption interface {
	ExtOption()
}

// ExtensionResult is an optional value that may be returned by
// backend.
//
// Unknown value types are ignored.
type ExtensionResult interface {
	ExtResult()
}
