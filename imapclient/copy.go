package imapclient

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Client) copy(uid bool, seqSet imap.SeqSet, mailbox string) *CopyCommand {
	cmd := &CopyCommand{}
	enc := c.beginCommand(uidCmdName("COPY", uid), cmd)
	enc.SP().Atom(seqSet.String()).SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Copy sends a COPY command.
func (c *Client) Copy(seqSet imap.SeqSet, mailbox string) *CopyCommand {
	return c.copy(false, seqSet, mailbox)
}

// UIDCopy sends a UID COPY command.
//
// See Copy.
func (c *Client) UIDCopy(seqSet imap.SeqSet, mailbox string) *CopyCommand {
	return c.copy(true, seqSet, mailbox)
}

// CopyCommand is a COPY command.
type CopyCommand struct {
	cmd
	data CopyData
}

func (cmd *CopyCommand) Wait() (*CopyData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// CopyData is the data returned by a COPY command.
type CopyData struct {
	// requires UIDPLUS or IMAP4rev2
	UIDValidity uint32
	SourceUIDs  imap.SeqSet
	DestUIDs    imap.SeqSet
}

func readRespCodeCopy(dec *imapwire.Decoder) (uidValidity uint32, srcUIDs, dstUIDs imap.SeqSet, err error) {
	var srcStr, dstStr string
	if !dec.ExpectNumber(&uidValidity) || !dec.ExpectSP() || !dec.ExpectAtom(&srcStr) || !dec.ExpectSP() || !dec.ExpectAtom(&dstStr) {
		return 0, imap.SeqSet{}, imap.SeqSet{}, dec.Err()
	}
	srcUIDs, err = imap.ParseSeqSet(srcStr)
	if err != nil {
		return 0, imap.SeqSet{}, imap.SeqSet{}, dec.Err()
	}
	dstUIDs, err = imap.ParseSeqSet(dstStr)
	if err != nil {
		return 0, imap.SeqSet{}, imap.SeqSet{}, dec.Err()
	}
	return uidValidity, srcUIDs, dstUIDs, nil
}
