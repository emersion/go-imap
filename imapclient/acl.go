package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// MyRights sends a MYRIGHTS command.
func (c *Client) MyRights(mailbox string) *MyRightsCommand {
	cmd := &MyRightsCommand{}
	enc := c.beginCommand("MYRIGHTS", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
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
	data imap.MyRightsData
}

func (cmd *MyRightsCommand) Wait() (*imap.MyRightsData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

func readMyRights(dec *imapwire.Decoder) (*imap.MyRightsData, error) {
	var data imap.MyRightsData
	var rights string

	if !dec.ExpectMailbox(&data.Mailbox) || !dec.ExpectSP() || !dec.ExpectAString(&rights) {
		return nil, dec.Err()
	}

	_, rs, err := imap.NewRights(rights, true)
	if err != nil {
		return nil, err
	}

	data.Rights = rs

	return &data, nil
}

// SetACL sends a SETACL command.
func (c *Client) SetACL(
	mailbox string, ri imap.RightsIdentifier, rm imap.RightModification, rs imap.RightSet,
) *SetACLCommand {
	cmd := &SetACLCommand{}
	enc := c.beginCommand("SETACL", cmd)
	enc.SP().Mailbox(mailbox).SP().String(string(ri)).SP()

	rightsStr := string(rs)
	if rm != imap.RightModificationReplace {
		rightsStr = string(rm) + rightsStr
	}

	enc.String(rightsStr)
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

// GetACL sends a GETACL command.
func (c *Client) GetACL(mailbox string) *GetACLCommand {
	cmd := &GetACLCommand{}
	enc := c.beginCommand("GETACL", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

func (c *Client) handleGetACL() error {
	data, err := readGetACL(c.dec)
	if err != nil {
		return fmt.Errorf("in getacl-response: %v", err)
	}
	if cmd := findPendingCmdByType[*GetACLCommand](c); cmd != nil {
		cmd.data = *data
	}
	return nil
}

// GetACLCommand is a GETACL command.
type GetACLCommand struct {
	cmd
	data imap.GetACLData
}

func readGetACL(dec *imapwire.Decoder) (*imap.GetACLData, error) {
	data := &imap.GetACLData{Rights: make(map[imap.RightsIdentifier]imap.RightSet)}

	if !dec.ExpectMailbox(&data.Mailbox) {
		return nil, dec.Err()
	}

	for dec.SP() {
		var rsStr, riStr string

		if !dec.ExpectAString(&riStr) || !dec.ExpectSP() || !dec.ExpectAString(&rsStr) {
			return nil, dec.Err()
		}

		_, rs, err := imap.NewRights(rsStr, true)
		if err != nil {
			return nil, err
		}

		data.Rights[imap.RightsIdentifier(riStr)] = rs
	}

	return data, nil
}

func (cmd *GetACLCommand) Wait() (*imap.GetACLData, error) {
	return &cmd.data, cmd.cmd.Wait()
}
