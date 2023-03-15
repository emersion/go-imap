package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

func (c *Client) move(uid bool, seqSet imap.SeqSet, mailbox string) *MoveCommand {
	cmd := &MoveCommand{}
	enc := c.beginCommand(uidCmdName("MOVE", uid), cmd)
	enc.SP().Atom(seqSet.String()).SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Move sends a MOVE command.
func (c *Client) Move(seqSet imap.SeqSet, mailbox string) *MoveCommand {
	return c.move(false, seqSet, mailbox)
}

// UIDMove sends a UID MOVE command.
//
// See Move.
func (c *Client) UIDMove(seqSet imap.SeqSet, mailbox string) *MoveCommand {
	return c.move(true, seqSet, mailbox)
}

// MoveCommand is a MOVE command.
type MoveCommand struct {
	cmd
	data MoveData
}

func (cmd *MoveCommand) Wait() (*MoveData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// MoveData contains the data returned by a MOVE command.
type MoveData struct {
	// requires UIDPLUS or IMAP4rev2
	UIDValidity uint32
	SourceUIDs  imap.SeqSet
	DestUIDs    imap.SeqSet
}
