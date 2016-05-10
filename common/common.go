// Generic structures and functions for IMAP.
package common

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
