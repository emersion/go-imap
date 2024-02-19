package imapclient

import "github.com/opsxolc/go-imap/v2"

// SetACL sends a SETACL command.
func (c *Client) SetACL(mailbox string, identifier imap.RightsIdentifier, rights imap.RightSet) *SetACLCommand {
	cmd := &SetACLCommand{}
	enc := c.beginCommand("SETACL", cmd)
	enc.SP().Mailbox(mailbox).SP().String(string(identifier)).SP().String(string(rights))
	enc.end()
	return cmd
}

// SetACLCommand is a SETACL command.
type SetACLCommand struct {
	cmd
}

func (cmd *SetACLCommand) Wait() error {
	return cmd.cmd.Wait()
}
