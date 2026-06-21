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

// handleVanished parses a VANISHED response (RFC 7162) and delivers the expunged
// UIDs to the unilateral VANISHED handler. The grammar is:
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
		earlier = strings.EqualFold(name, "EARLIER")
	}
	var uids imap.UIDSet
	if !c.dec.ExpectUIDSet(&uids) {
		return c.dec.Err()
	}
	if handler := c.options.unilateralDataHandler().Vanished; handler != nil {
		handler(uids, earlier)
	}
	return nil
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

// ExpungeCommand is an EXPUNGE command.
//
// The caller must fully consume the ExpungeCommand. A simple way to do so is
// to defer a call to FetchCommand.Close.
type ExpungeCommand struct {
	commandBase
	seqNums chan uint32
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
