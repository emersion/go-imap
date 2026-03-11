package imap

// VanishedData represents a VANISHED response.
type VanishedData struct {
	// Earlier indicates this is a VANISHED (EARLIER) response
	// sent during SELECT with QRESYNC
	Earlier bool
	// UIDs is the set of UIDs that have been expunged
	UIDs UIDSet
}
