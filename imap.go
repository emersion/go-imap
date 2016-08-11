// An IMAP4rev1 (RFC 3501) library written in Go. It can be used to build a
// client and/or a server and supports UTF-7.
package imap

// An operation that will be applied on message flags.
type FlagsOp string

const (
	// Replace existing flags by new ones
	SetFlags FlagsOp = "FLAGS"
	// Add new flags
	AddFlags = "+FLAGS"
	// Remove existing flags
	RemoveFlags = "-FLAGS"
)

const SilentOp = ".SILENT"
