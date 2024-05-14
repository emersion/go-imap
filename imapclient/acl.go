package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// MyRights sends a MYRIGHTS command.
//
// This command requires support for the ACL extension.
func (c *Client) MyRights(mailbox string) *MyRightsCommand {
	cmd := &MyRightsCommand{}
	enc := c.beginCommand("MYRIGHTS", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// SetACL sends a SETACL command.
//
// This command requires support for the ACL extension.
func (c *Client) SetACL(mailbox string, ri imap.RightsIdentifier, rm imap.RightModification, rs imap.RightSet) *SetACLCommand {
	cmd := &SetACLCommand{}
	enc := c.beginCommand("SETACL", cmd)
	enc.SP().Mailbox(mailbox).SP().String(string(ri)).SP()
	enc.String(internal.FormatRights(rm, rs))
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

func (c *Client) handleMyRights() error {
	data, err := readMyRights(c.dec)
	if err != nil {
		return fmt.Errorf("in myrights-response: %v", err)
	}
	if cmd := findPendingCmdByType[*MyRightsCommand](c); cmd != nil {
		cmd.data = *data
	}
	return nil
}

// MyRightsCommand is a MYRIGHTS command.
type MyRightsCommand struct {
	cmd
	data MyRightsData
}

func (cmd *MyRightsCommand) Wait() (*MyRightsData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// MyRightsData is the data returned by the MYRIGHTS command.
type MyRightsData struct {
	Mailbox string
	Rights  imap.RightSet
}

func readMyRights(dec *imapwire.Decoder) (*MyRightsData, error) {
	var (
		rights string
		data   MyRightsData
	)
	if !dec.ExpectMailbox(&data.Mailbox) || !dec.ExpectSP() || !dec.ExpectAString(&rights) {
		return nil, dec.Err()
	}

	data.Rights = imap.RightSet(rights)
	return &data, nil
}
