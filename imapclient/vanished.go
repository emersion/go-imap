package imapclient

import (
	"strings"

	"github.com/emersion/go-imap/v2"
)

// handleVanished parses a VANISHED response (RFC 7162) and either collects the
// expunged UIDs on a pending SELECT/EXAMINE command (QRESYNC resynchronization)
// or delivers them to the unilateral VANISHED handler. The grammar is:
//
//	"VANISHED" [SP "(EARLIER)"] SP known-uids
func (c *Client) handleVanished() error {
	if !c.dec.ExpectSP() {
		return c.dec.Err()
	}
	var earlier bool
	if c.dec.Special('(') {
		var name string
		if !c.dec.ExpectAtom(&name) || !c.dec.ExpectSpecial(')') || !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		// EARLIER is the only fetch-modifier defined for VANISHED. An unknown
		// modifier is still recoverable (the remainder is always SP known-uids),
		// so tolerate it rather than tearing down the connection, matching how
		// unknown resp-text-codes are discarded.
		earlier = strings.EqualFold(name, "EARLIER")
	}
	var uids imap.UIDSet
	if !c.dec.ExpectUIDSet(&uids) {
		return c.dec.Err()
	}

	// A VANISHED response arriving while a QRESYNC SELECT/EXAMINE is in progress
	// is part of the resynchronization (always VANISHED (EARLIER) in practice);
	// collect it on the command instead of treating it as unilateral data.
	if cmd := findPendingCmdByType[*SelectCommand](c); cmd != nil {
		cmd.data.VanishedUIDs = append(cmd.data.VanishedUIDs, uids...)
		return nil
	}
	if handler := c.options.unilateralDataHandler().Vanished; handler != nil {
		handler(uids, earlier)
	}
	return nil
}
