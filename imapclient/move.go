package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

func (c *Client) move(uid bool, seqSet imap.SeqSet, mailbox string) *ExpungeCommand {
	cmd := &ExpungeCommand{seqNums: make(chan uint32, 128)}
	enc := c.beginCommand(uidCmdName("MOVE", uid), cmd)
	enc.SP().Atom(seqSet.String()).SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Move sends a MOVE command.
func (c *Client) Move(seqSet imap.SeqSet, mailbox string) *ExpungeCommand {
	return c.move(false, seqSet, mailbox)
}

// UIDMove sends a UID MOVE command.
//
// See Move.
func (c *Client) UIDMove(seqSet imap.SeqSet, mailbox string) *ExpungeCommand {
	return c.move(true, seqSet, mailbox)
}
