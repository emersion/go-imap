package client

import (
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
)

// TODO: CHECK

func (c *Client) Close() (err error) {
	if c.State != imap.SelectedState {
		err = errors.New("No mailbox selected")
		return
	}

	cmd := &commands.Close{}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}

	err = status.Err()
	if err != nil {
		return
	}

	c.State = imap.AuthenticatedState
	c.Selected = nil
	return
}

// TODO: EXPUNGE, SEARCH

func (c *Client) Fetch(seqset *imap.SeqSet, items []string, ch chan<- *imap.Message) (err error) {
	defer close(ch)

	if c.State != imap.SelectedState {
		err = errors.New("No mailbox selected")
		return
	}

	cmd := &commands.Fetch{
		SeqSet: seqset,
		Items: items,
	}
	res := &responses.Fetch{Messages: ch}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

// TODO: STORE, COPY, UID
