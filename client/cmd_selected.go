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

func (c *Client) Expunge(ch chan<- uint32) (err error) {
	defer close(ch)

	if c.State != imap.SelectedState {
		err = errors.New("No mailbox selected")
		return
	}

	cmd := &commands.Expunge{}

	var res *responses.Expunge
	if ch != nil {
		res = &responses.Expunge{SeqIds: ch}
	}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}

	err = status.Err()
	return
}

func (c *Client) Search(criteria []interface{}) (ids []uint32, err error) {
	if c.State != imap.SelectedState {
		err = errors.New("No mailbox selected")
		return
	}

	cmd := &commands.Search{
		Charset: "UTF-8",
		Criteria: criteria,
	}

	res := &responses.Search{}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}

	err = status.Err()
	ids = res.Ids
	return
}

// TODO: SEARCH

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
