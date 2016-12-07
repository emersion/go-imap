package client

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
)

var ErrAlreadyLoggedOut = errors.New("Already logged out")

// Request a listing of capabilities that the server supports. Capabilities are
// often returned by the server with the greeting or with the STARTTLS and LOGIN
// responses, so usually explicitly requesting capabilities isn't needed.
func (c *Client) Capability() (caps map[string]bool, err error) {
	cmd := &commands.Capability{}
	res := &responses.Capability{}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}

	err = status.Err()
	if err != nil {
		return
	}

	caps = map[string]bool{}
	for _, name := range res.Caps {
		caps[name] = true
	}

	c.Caps = caps
	return
}

// This command always succeeds. It does nothing.
// Can be used as a periodic poll for new messages or message status updates
// during a period of inactivity. Can also be used to reset any inactivity
// autologout timer on the server.
func (c *Client) Noop() (err error) {
	cmd := &commands.Noop{}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// Close the connection.
func (c *Client) Logout() error {
	if c.State == imap.LogoutState {
		return ErrAlreadyLoggedOut
	}

	cmd := &commands.Logout{}

	if status, err := c.execute(cmd, nil); err == errClosed {
		// Server closed connection, that's what we want anyway
		return nil
	} else if err != nil {
		return err
	} else if status != nil {
		return status.Err()
	}
	return nil
}
