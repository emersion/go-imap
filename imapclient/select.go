package imapclient

import (
	"github.com/emersion/go-imap/v2"
)

// Select sends a SELECT command.
func (c *Client) Select(mailbox string) *SelectCommand {
	cmd := &SelectCommand{mailbox: mailbox}
	enc := c.beginCommand("SELECT", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Examine sends an EXAMINE command.
//
// See Select.
func (c *Client) Examine(mailbox string) *SelectCommand {
	cmd := &SelectCommand{mailbox: mailbox}
	enc := c.beginCommand("EXAMINE", cmd)
	enc.SP().Mailbox(mailbox)
	enc.end()
	return cmd
}

// Unselect sends an UNSELECT command.
//
// This command requires support for IMAP4rev2 or the UNSELECT extension.
func (c *Client) Unselect() *Command {
	cmd := &unselectCommand{}
	c.beginCommand("UNSELECT", cmd).end()
	return &cmd.cmd
}

// UnselectAndExpunge sends a CLOSE command.
//
// CLOSE implicitly performs a silent EXPUNGE command.
func (c *Client) UnselectAndExpunge() *Command {
	cmd := &unselectCommand{}
	c.beginCommand("CLOSE", cmd).end()
	return &cmd.cmd
}

func (c *Client) handleFlags() error {
	flags, err := readFlagList(c.dec)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	if c.state == StateSelected {
		c.mailbox = c.mailbox.copy()
		c.mailbox.PermanentFlags = flags
	}
	c.mutex.Unlock()

	cmd := findPendingCmdByType[*SelectCommand](c)
	if cmd != nil {
		cmd.data.Flags = flags
	} else if handler := c.options.unilateralDataHandler().Mailbox; handler != nil {
		handler(&UnilateralDataMailbox{Flags: flags})
	}

	return nil
}

func (c *Client) handleExists(num uint32) error {
	cmd := findPendingCmdByType[*SelectCommand](c)
	if cmd != nil {
		cmd.data.NumMessages = num
	} else {
		c.mutex.Lock()
		if c.state == StateSelected {
			c.mailbox = c.mailbox.copy()
			c.mailbox.NumMessages = num
		}
		c.mutex.Unlock()

		if handler := c.options.unilateralDataHandler().Mailbox; handler != nil {
			handler(&UnilateralDataMailbox{NumMessages: &num})
		}
	}
	return nil
}

// SelectCommand is a SELECT command.
type SelectCommand struct {
	cmd
	mailbox string
	data    SelectData
}

func (cmd *SelectCommand) Wait() (*SelectData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

type unselectCommand struct {
	cmd
}

// SelectData is the data returned by a SELECT command.
//
// In the old RFC 2060, PermanentFlags, UIDNext and UIDValidity are optional.
type SelectData struct {
	// Flags defined for this mailbox
	Flags []imap.Flag
	// Flags that the client can change permanently
	PermanentFlags []imap.Flag
	// Number of messages in this mailbox (aka. "EXISTS")
	NumMessages uint32
	UIDNext     uint32
	UIDValidity uint32

	List *ListData // requires IMAP4rev2
}
