package client

import (
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
)

// Selects a mailbox so that messages in the mailbox can be accessed. Any
// currently selected mailbox is deselected before attempting the new selection.
// Even if the readOnly parameter is set to false, the server can decide to open
// the mailbox in read-only mode.
func (c *Client) Select(name string, readOnly bool) (mbox *imap.MailboxStatus, err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = errors.New("Not logged in")
		return
	}

	mbox = &imap.MailboxStatus{Name: name}

	cmd := &commands.Select{
		Mailbox: name,
		ReadOnly: readOnly,
	}
	res := &responses.Select{
		Mailbox: mbox,
	}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}

	err = status.Err()
	if err != nil {
		return
	}

	mbox.ReadOnly = (status.Code == "READ-ONLY")

	c.State = imap.SelectedState
	c.Selected = mbox
	return
}

// TODO: CREATE, DELETE, RENAME, SUBSCRIBE, UNSUBSCRIBE

// Returns a subset of names from the complete set of all names available to the
// client.
// An empty name argument is a special request to return the hierarchy delimiter
// and the root name of the name given in the reference. The character "*" is a
// wildcard, and matches zero or more characters at this position. The
// character "%" is similar to "*", but it does not match a hierarchy delimiter.
func (c *Client) List(ref, name string, ch chan<- *imap.MailboxInfo) (err error) {
	defer close(ch)

	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = errors.New("Not logged in")
		return
	}

	cmd := &commands.List{
		Reference: ref,
		Mailbox: name,
	}
	res := &responses.List{Mailboxes: ch}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// TODO: LSUB

// Requests the status of the indicated mailbox. It does not change the
// currently selected mailbox, nor does it affect the state of any messages in
// the queried mailbox. See RFC 2501 section 6.3.10 for a list of items that can
// be requested.
func (c *Client) Status(name string, items []string) (mbox *imap.MailboxStatus, err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = errors.New("Not logged in")
		return
	}

	mbox = &imap.MailboxStatus{}

	cmd := &commands.Status{
		Mailbox: name,
		Items: items,
	}
	res := &responses.Status{
		Mailbox: mbox,
	}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// TODO: APPEND
