package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Myrights sends a MYRIGHTS command.
func (c *Client) Myrights(mailbox string) *MyrightsCommand {
	cmd := &MyrightsCommand{mailbox: mailbox}
	enc := c.beginCommand("MYRIGHTS", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

func (c *Client) handleMyrights() error {
	data, err := readMyrights(c.dec)
	if err != nil {
		return fmt.Errorf("in myrights-response: %v", err)
	}
	if cmd := findPendingCmdByType[*MyrightsCommand](c); cmd != nil {
		cmd.data = *data
	}
	return nil
}

// MyrightsCommand is a MYRIGHTS command.
type MyrightsCommand struct {
	cmd
	mailbox string
	data    imap.MyrightsData
}

func (cmd *MyrightsCommand) Wait() (*imap.MyrightsData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

func readMyrights(dec *imapwire.Decoder) (*imap.MyrightsData, error) {
	var data imap.MyrightsData

	if !dec.ExpectMailbox(&data.Mailbox) || !dec.ExpectSP() || !dec.ExpectText(&data.Rights) {
		return nil, dec.Err()
	}

	return &data, nil
}
