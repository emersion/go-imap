package client

import (
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
)

func (c *Client) Capability() (caps map[string]bool, err error) {
	cmd := &commands.Capability{}

	_, err = c.execute(cmd, nil)
	caps = c.Caps
	return
}

// TODO: NOOP

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

	err = status.Err()
	return
}
