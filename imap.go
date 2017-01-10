// Package imap implements IMAP4rev1 (RFC 3501).
package imap

import (
	"io"
)

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

// CharsetReader, if non-nil, defines a function to generate charset-conversion
// readers, converting from the provided charset into UTF-8. Charsets are always
// lower-case. utf-8 and us-ascii charsets are handled by default. One of the
// the CharsetReader's result values must be non-nil.
var CharsetReader func(charset string, r io.Reader) (io.Reader, error)
