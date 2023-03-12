package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

func (c *Client) copy(uid bool, seqSet imap.SeqSet, mailbox string) *Command {
	cmd := &Command{}
	enc := c.beginCommand(uidCmdName("COPY", uid), cmd)
	enc.SP().Atom(seqSet.String()).SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Copy sends a COPY command.
func (c *Client) Copy(seqSet imap.SeqSet, mailbox string) *Command {
	return c.copy(false, seqSet, mailbox)
}

// UIDCopy sends a UID COPY command.
//
// See Copy.
func (c *Client) UIDCopy(seqSet imap.SeqSet, mailbox string) *Command {
	return c.copy(true, seqSet, mailbox)
}
