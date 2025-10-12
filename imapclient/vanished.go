package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

func (c *Client) handleVanished() error {
	var earlier bool
	if c.dec.Special('(') {
		var atom string
		if !c.dec.ExpectAtom(&atom) || atom != "EARLIER" || !c.dec.ExpectSpecial(')') {
			return c.dec.Err()
		}
		earlier = true
		if !c.dec.ExpectSP() {
			return c.dec.Err()
		}
	}

	var uids imap.UIDSet
	if !c.dec.ExpectUIDSet(&uids) {
		return c.dec.Err()
	}

	// Check if this is part of a SELECT command response
	cmd := findPendingCmdByType[*SelectCommand](c)
	if cmd != nil {
		cmd.data.VanishedUIDs = uids
	} else if handler := c.options.unilateralDataHandler().Vanished; handler != nil {
		handler(uids, earlier)
	}

	return nil
}
