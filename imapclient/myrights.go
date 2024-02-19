package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// MyRights sends a MYRIGHTS command.
func (c *Client) MyRights(mailbox string) *MyRightsCommand {
	cmd := &MyRightsCommand{mailbox: mailbox}
	enc := c.beginCommand("MYRIGHTS", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

func (c *Client) handleMyrights() error {
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
	mailbox string
	data    imap.MyRightsData
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

	rightSet, err := imap.NewRightSet(rights)
	if err != nil {
		return nil, err
	}

	data.Rights = rightSet

	return &data, nil
}
