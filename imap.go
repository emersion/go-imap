// Package imap implements IMAP4rev1 (RFC 3501).
package imap

// FlagsOp is an operation that will be applied on message flags.
type FlagsOp string

const (
	// SetFlags replaces existing flags by new ones.
	SetFlags FlagsOp = "FLAGS"
	// AddFlags adds new flags.
	AddFlags = "+FLAGS"
	// RemoveFlags removes existing flags.
	RemoveFlags = "-FLAGS"
)

// SilentOp can be appended to a FlagsOp to prevent the operation from
// triggering unilateral message updates.
const SilentOp = ".SILENT"
