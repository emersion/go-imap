package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

func (c *Client) move(uid bool, seqSet imap.NumSet, mailbox string) *MoveCommand {
	// If the server doesn't support MOVE, fallback to [UID] COPY,
	// [UID] STORE +FLAGS.SILENT \Deleted and [UID] EXPUNGE
	cmdName := "MOVE"
	if !c.Caps().Has(imap.CapMove) {
		cmdName = "COPY"
	}

	cmd := &MoveCommand{}
	enc := c.beginCommand(uidCmdName(cmdName, uid), cmd)
	enc.SP().NumSet(seqSet).SP().Mailbox(mailbox)
	enc.end()

	if cmdName == "COPY" {
		cmd.store = c.store(uid, seqSet, &imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagDeleted},
		}, nil)
		if uid && c.Caps().Has(imap.CapUIDPlus) {
			cmd.expunge = c.UIDExpunge(seqSet)
		} else {
			cmd.expunge = c.Expunge()
		}
	}

	return cmd
}

// Move sends a MOVE command.
//
// If the server doesn't support IMAP4rev2 nor the MOVE extension, a fallback
// with COPY + STORE + EXPUNGE commands is used.
func (c *Client) Move(seqSet imap.NumSet, mailbox string) *MoveCommand {
	return c.move(false, seqSet, mailbox)
}

// UIDMove sends a UID MOVE command.
//
// See Move.
func (c *Client) UIDMove(seqSet imap.NumSet, mailbox string) *MoveCommand {
	return c.move(true, seqSet, mailbox)
}

// MoveCommand is a MOVE command.
type MoveCommand struct {
	cmd
	data MoveData

	// Fallback
	store   *FetchCommand
	expunge *ExpungeCommand
}

func (cmd *MoveCommand) Wait() (*MoveData, error) {
	if err := cmd.cmd.Wait(); err != nil {
		return nil, err
	}
	if cmd.store != nil {
		if err := cmd.store.Close(); err != nil {
			return nil, err
		}
	}
	if cmd.expunge != nil {
		if err := cmd.expunge.Close(); err != nil {
			return nil, err
		}
	}
	return &cmd.data, nil
}

// MoveData contains the data returned by a MOVE command.
type MoveData struct {
	// requires UIDPLUS or IMAP4rev2
	UIDValidity uint32
	SourceUIDs  imap.NumSet
	DestUIDs    imap.NumSet
}
