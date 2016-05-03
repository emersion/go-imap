package client

import (
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
)

// Request a listing of capabilities that the server supports. Capabilities are
// often returned by the server with the greeting or with the STARTTLS and LOGIN
// responses, so usally explicitely requesting capabilities isn't needed.
func (c *Client) Capability() (caps map[string]bool, err error) {
	cmd := &commands.Capability{}

	_, err = c.execute(cmd, nil)
	caps = c.Caps
	return
}

// TODO: NOOP

// Close the connection.
func (c *Client) Logout() (err error) {
	if c.State == imap.LogoutState {
		err = errors.New("Already logged out")
		return
	}

	cmd := &commands.Logout{}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	if status != nil {
		err = status.Err()
	}
	return
}
