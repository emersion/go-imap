package imapclient

import (
	"fmt"
	"unicode/utf8"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// List sends a LIST command.
//
// The caller must fully consume the ListCommand. A simple way to do so is to
// defer a call to ListCommand.Close.
func (c *Client) List(ref, pattern string) *ListCommand {
	// TODO: extended variant
	cmd := &ListCommand{mailboxes: make(chan *ListData, 64)}
	enc := c.beginCommand("LIST", cmd)
	enc.SP().Mailbox(ref).SP().String(pattern)
	enc.end()
	return cmd
}

// ListCommand is a LIST command.
type ListCommand struct {
	cmd
	mailboxes chan *ListData
}

// Next advances to the next mailbox.
//
// On success, the mailbox LIST data is returned. On error or if there are no
// more mailboxes, nil is returned.
func (cmd *ListCommand) Next() *ListData {
	return <-cmd.mailboxes
}

// Close releases the command.
//
// Calling Close unblocks the IMAP client decoder and lets it read the next
// responses. Next will always return nil after Close.
func (cmd *ListCommand) Close() error {
	for cmd.Next() != nil {
		// ignore
	}
	return cmd.cmd.Wait()
}

// Collect accumulates mailboxes into a list.
//
// This is equivalent to calling Next repeatedly and then Close.
func (cmd *ListCommand) Collect() ([]*ListData, error) {
	var l []*ListData
	for {
		data := cmd.Next()
		if data == nil {
			break
		}
		l = append(l, data)
	}
	return l, cmd.Close()
}

// ListData is the mailbox data returned by a LIST command.
type ListData struct {
	Attrs   []imap.MailboxAttr
	Delim   rune
	Mailbox string
}

func readList(dec *imapwire.Decoder) (*ListData, error) {
	var data ListData

	err := dec.ExpectList(func() error {
		attr, err := readFlag(dec)
		if err != nil {
			return err
		}
		data.Attrs = append(data.Attrs, imap.MailboxAttr(attr))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("in mbx-list-flags")
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	var delimStr string
	if dec.Quoted(&delimStr) {
		delim, size := utf8.DecodeRuneInString(delimStr)
		if delim == utf8.RuneError || size != len(delimStr) {
			return nil, fmt.Errorf("mailbox delimiter must be a single rune")
		}
		data.Delim = delim
	} else if !dec.ExpectNIL() {
		return nil, dec.Err()
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	if data.Mailbox, err = dec.ExpectMailbox(); err != nil {
		return nil, err
	}

	// TODO: [SP mbox-list-extended]

	return &data, nil
}
