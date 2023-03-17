package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

// Enable sends an ENABLE command.
//
// This command requires support for IMAP4rev2 or the ENABLE extension.
func (c *Client) Enable(caps ...imap.Cap) *EnableCommand {
	cmd := &EnableCommand{}
	enc := c.beginCommand("ENABLE", cmd)
	for _, c := range caps {
		enc.SP().Atom(string(c))
	}
	enc.end()
	return cmd
}

func (c *Client) handleEnabled() error {
	caps, err := readCapabilities(c.dec)
	if err != nil {
		return err
	}
	if cmd := findPendingCmdByType[*EnableCommand](c); cmd != nil {
		cmd.data.Caps = caps
	}
	return nil
}

// EnableCommand is an ENABLE command.
type EnableCommand struct {
	cmd
	data EnableData
}

func (cmd *EnableCommand) Wait() (*EnableData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// EnableData is the data returned by the ENABLE command.
type EnableData struct {
	// Capabilities that were successfully enabled
	Caps imap.CapSet
}
