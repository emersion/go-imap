package client

import (
	"errors"
	"time"

	imap "github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
)

var ErrNotLoggedIn = errors.New("Not logged in")

// Selects a mailbox so that messages in the mailbox can be accessed. Any
// currently selected mailbox is deselected before attempting the new selection.
// Even if the readOnly parameter is set to false, the server can decide to open
// the mailbox in read-only mode.
func (c *Client) Select(name string, readOnly bool) (mbox *imap.MailboxStatus, err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
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

	c.Mailbox = mbox

	status, err := c.execute(cmd, res)
	if err != nil {
		c.Mailbox = nil
		return
	}

	err = status.Err()
	if err != nil {
		c.Mailbox = nil
		return
	}

	mbox.ReadOnly = (status.Code == "READ-ONLY")
	c.State = imap.SelectedState
	return
}

// Creates a mailbox with the given name.
func (c *Client) Create(name string) (err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
		return
	}

	cmd := &commands.Create{
		Mailbox: name,
	}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// Permanently removes the mailbox with the given name.
func (c *Client) Delete(name string) (err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
		return
	}

	cmd := &commands.Delete{
		Mailbox: name,
	}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// Changes the name of a mailbox.
func (c *Client) Rename(existingName, newName string) (err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
		return
	}

	cmd := &commands.Rename{
		Existing: existingName,
		New: newName,
	}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// Adds the specified mailbox name to the server's set of "active" or
// "subscribed" mailboxes.
func (c *Client) Subscribe(name string) (err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
		return
	}

	cmd := &commands.Subscribe{
		Mailbox: name,
	}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// Removes the specified mailbox name from the server's set of "active" or
// "subscribed" mailboxes.
func (c *Client) Unsubscribe(name string) (err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
		return
	}

	cmd := &commands.Unsubscribe{
		Mailbox: name,
	}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// Returns a subset of names from the complete set of all names available to the
// client.
// An empty name argument is a special request to return the hierarchy delimiter
// and the root name of the name given in the reference. The character "*" is a
// wildcard, and matches zero or more characters at this position. The
// character "%" is similar to "*", but it does not match a hierarchy delimiter.
func (c *Client) List(ref, name string, ch chan *imap.MailboxInfo) (err error) {
	defer close(ch)

	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
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

// Returns a subset of names from the set of names that the user has declared as
// being "active" or "subscribed".
func (c *Client) Lsub(ref, name string, ch chan *imap.MailboxInfo) (err error) {
	defer close(ch)

	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
		return
	}

	cmd := &commands.List{
		Reference: ref,
		Mailbox: name,
		Subscribed: true,
	}
	res := &responses.List{
		Mailboxes: ch,
		Subscribed: true,
	}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// Requests the status of the indicated mailbox. It does not change the
// currently selected mailbox, nor does it affect the state of any messages in
// the queried mailbox.
// See RFC 2501 section 6.3.10 for a list of items that can be requested.
func (c *Client) Status(name string, items []string) (mbox *imap.MailboxStatus, err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
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

// Appends the literal argument as a new message to the end of the specified
// destination mailbox. This argument SHOULD be in the format of an RFC 2822
// message.
// flags and date are optional arguments and can be set to nil.
func (c *Client) Append(mbox string, flags []string, date time.Time, msg *imap.Literal) (err error) {
	if c.State != imap.AuthenticatedState && c.State != imap.SelectedState {
		err = ErrNotLoggedIn
		return
	}

	cmd := &commands.Append{
		Mailbox: mbox,
		Flags: flags,
		Date: date,
		Message: msg,
	}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	err = status.Err()
	return
}
