package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

// Select sends a SELECT command.
func (c *Client) Select(mailbox string) *SelectCommand {
	cmd := &SelectCommand{}
	enc := c.beginCommand("SELECT", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Examine sends an EXAMINE command.
//
// See Select.
func (c *Client) Examine(mailbox string) *SelectCommand {
	cmd := &SelectCommand{}
	enc := c.beginCommand("EXAMINE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Unselect sends an UNSELECT command.
func (c *Client) Unselect() *Command {
	cmd := &Command{}
	c.beginCommand("UNSELECT", cmd).end()
	return cmd
}

// UnselectAndExpunge sends a CLOSE command.
//
// CLOSE implicitly performs a silent EXPUNGE command.
func (c *Client) UnselectAndExpunge() *Command {
	cmd := &Command{}
	c.beginCommand("CLOSE", cmd).end()
	return cmd
}

// SelectCommand is a SELECT command.
type SelectCommand struct {
	cmd
	data SelectData
}

func (cmd *SelectCommand) Wait() (*SelectData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// SelectData is the data returned by a SELECT command.
type SelectData struct {
	// Flags defined for this mailbox
	Flags []imap.Flag
	// Number of messages in this mailbox (aka. "EXISTS")
	NumMessages uint32

	// TODO: LIST, PERMANENTFLAGS, UIDNEXT, UIDVALIDITY
}
