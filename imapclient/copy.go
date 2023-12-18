package imapclient

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Client) copy(uid bool, seqSet imap.NumSet, mailbox string) *CopyCommand {
	cmd := &CopyCommand{}
	enc := c.beginCommand(uidCmdName("COPY", uid), cmd)
	enc.SP().NumSet(seqSet).SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Copy sends a COPY command.
func (c *Client) Copy(seqSet imap.NumSet, mailbox string) *CopyCommand {
	return c.copy(false, seqSet, mailbox)
}

// UIDCopy sends a UID COPY command.
//
// See Copy.
func (c *Client) UIDCopy(seqSet imap.NumSet, mailbox string) *CopyCommand {
	return c.copy(true, seqSet, mailbox)
}

// CopyCommand is a COPY command.
type CopyCommand struct {
	cmd
	data imap.CopyData
}

func (cmd *CopyCommand) Wait() (*imap.CopyData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

func readRespCodeCopy(dec *imapwire.Decoder) (uidValidity uint32, srcUIDs, dstUIDs imap.NumSet, err error) {
	if !dec.ExpectNumber(&uidValidity) || !dec.ExpectSP() || !dec.ExpectNumSet(&srcUIDs) || !dec.ExpectSP() || !dec.ExpectNumSet(&dstUIDs) {
		return 0, imap.NumSet{}, imap.NumSet{}, dec.Err()
	}
	return uidValidity, srcUIDs, dstUIDs, nil
}
