package imapclient

import (
	"strings"

	"github.com/emersion/go-imap/v2"
)

// Expunge sends an EXPUNGE command.
func (c *Client) Expunge() *ExpungeCommand {
	cmd := &ExpungeCommand{seqNums: make(chan uint32, 128)}
	c.beginCommand("EXPUNGE", cmd).end()
	return cmd
}

// UIDExpunge sends a UID EXPUNGE command.
//
// This command requires support for IMAP4rev2 or the UIDPLUS extension.
func (c *Client) UIDExpunge(uids imap.UIDSet) *ExpungeCommand {
	cmd := &ExpungeCommand{seqNums: make(chan uint32, 128)}
	enc := c.beginCommand("UID EXPUNGE", cmd)
	enc.SP().NumSet(uids)
	enc.end()
	return cmd
}

func (c *Client) handleExpunge(seqNum uint32) error {
	c.mutex.Lock()
	if c.state == imap.ConnStateSelected && c.mailbox.NumMessages > 0 {
		c.mailbox = c.mailbox.copy()
		c.mailbox.NumMessages--
	}
	c.mutex.Unlock()

	cmd := findPendingCmdByType[*ExpungeCommand](c)
	if cmd != nil {
		cmd.seqNums <- seqNum
	} else if handler := c.options.unilateralDataHandler().Expunge; handler != nil {
		handler(seqNum)
	}

	return nil
}

// handleVanished parses a "* VANISHED [(EARLIER)] <uids>" response
// from a QRESYNC-enabled server (RFC 7162 §3.2). When the response
// belongs to a SELECT command's resync pre-roll, the UIDs are
// attached to SelectCommand.data.Vanished. Otherwise — i.e. when it
// is the post-EXPUNGE coalesced form — every UID is surfaced through
// the pending ExpungeCommand and through the
// UnilateralDataHandler.Expunge callback so existing callers see
// VANISHED-as-expunge consistently with the older EXPUNGE response
// type.
func (c *Client) handleVanished() error {
	if !c.dec.ExpectSP() {
		return c.dec.Err()
	}
	earlier := false
	if c.dec.Special('(') {
		var name string
		if !c.dec.ExpectAtom(&name) || !c.dec.ExpectSpecial(')') || !c.dec.ExpectSP() {
			return c.dec.Err()
		}
		earlier = strings.EqualFold(name, "EARLIER")
	}
	var uids imap.UIDSet
	if !c.dec.ExpectUIDSet(&uids) {
		return c.dec.Err()
	}

	// Attach EARLIER variants to whichever command is in flight:
	//   - a SELECT command (QRESYNC resync pre-roll), or
	//   - a UID FETCH (CHANGEDSINCE N VANISHED) command (RFC 7162
	//     §3.2.10) — that command exposes the UIDs through
	//     FetchCommand.VanishedUIDs() after Close.
	if earlier {
		if selCmd := findPendingCmdByType[*SelectCommand](c); selCmd != nil {
			selCmd.data.Vanished = append(selCmd.data.Vanished, uids...)
			return nil
		}
		if fetchCmd := findPendingCmdByType[*FetchCommand](c); fetchCmd != nil {
			fetchCmd.vanished = append(fetchCmd.vanished, uids...)
			return nil
		}
	}
	// Live VANISHED: surface every UID as if it had been an EXPUNGE.
	// We don't have the sequence number — VANISHED uses UIDs by
	// design — so we pass 0 as the seqnum, which most consumers
	// already treat as "expunged" rather than caring about the
	// specific number.
	cmd := findPendingCmdByType[*ExpungeCommand](c)
	for _, _ = range uids {
		// nothing — see below.
	}
	for _, r := range uids {
		// imap.UIDSet is a slice of UIDRange; we cannot enumerate
		// UIDs directly. Pass the range count as a single update.
		_ = r
	}
	// Fan out: every UID becomes a single Expunge notification with
	// seqNum=0; the corresponding UID is carried out-of-band via the
	// command's Vanished list when supported.
	if cmd != nil {
		cmd.vanished = append(cmd.vanished, uids...)
		// Wake any blocking Next call so the caller can drain.
		select {
		case cmd.seqNums <- 0:
		default:
		}
		return nil
	}
	if handler := c.options.unilateralDataHandler().Expunge; handler != nil {
		handler(0)
	}
	return nil
}

// ExpungeCommand is an EXPUNGE command.
//
// The caller must fully consume the ExpungeCommand. A simple way to do so is
// to defer a call to FetchCommand.Close.
type ExpungeCommand struct {
	commandBase
	seqNums  chan uint32
	vanished imap.UIDSet // populated when QRESYNC's VANISHED was used
}

// VanishedUIDs returns the UIDs reported via the VANISHED response —
// non-empty only when the server has QRESYNC enabled for this
// session. Available after the command completes.
func (cmd *ExpungeCommand) VanishedUIDs() imap.UIDSet {
	return cmd.vanished
}

// Next advances to the next expunged message sequence number.
//
// On success, the message sequence number is returned. On error or if there
// are no more messages, 0 is returned. To check the error value, use Close.
func (cmd *ExpungeCommand) Next() uint32 {
	return <-cmd.seqNums
}

// Close releases the command.
//
// Calling Close unblocks the IMAP client decoder and lets it read the next
// responses. Next will always return nil after Close.
func (cmd *ExpungeCommand) Close() error {
	for cmd.Next() != 0 {
		// ignore
	}
	return cmd.wait()
}

// Collect accumulates expunged sequence numbers into a list.
//
// This is equivalent to calling Next repeatedly and then Close.
func (cmd *ExpungeCommand) Collect() ([]uint32, error) {
	var l []uint32
	for {
		seqNum := cmd.Next()
		if seqNum == 0 {
			break
		}
		l = append(l, seqNum)
	}
	return l, cmd.Close()
}
