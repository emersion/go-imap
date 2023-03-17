package imapclient

import (
	"fmt"
	"unicode/utf8"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// ListOptions contains options for the LIST command.
type ListOptions struct {
	SelectSubscribed     bool
	SelectRemote         bool
	SelectRecursiveMatch bool // requires SelectSubscribed to be set

	ReturnSubscribed bool
	ReturnChildren   bool
	ReturnStatus     []StatusItem // requires IMAP4rev2 or LIST-STATUS
}

func (options *ListOptions) selectOpts() []string {
	if options == nil {
		return nil
	}

	var l []string
	if options.SelectSubscribed {
		l = append(l, "SUBSCRIBED")
	}
	if options.SelectRemote {
		l = append(l, "REMOTE")
	}
	if options.SelectRecursiveMatch {
		l = append(l, "RECURSIVEMATCH")
	}
	return l
}

func (options *ListOptions) returnOpts() []string {
	if options == nil {
		return nil
	}

	var l []string
	if options.ReturnSubscribed {
		l = append(l, "SUBSCRIBED")
	}
	if options.ReturnChildren {
		l = append(l, "CHILDREN")
	}
	if len(options.ReturnStatus) > 0 {
		l = append(l, "STATUS")
	}
	return l
}

// List sends a LIST command.
//
// The caller must fully consume the ListCommand. A simple way to do so is to
// defer a call to ListCommand.Close.
//
// A nil options pointer is equivalent to a zero options value.
//
// A non-zero options value requires support for IMAP4rev2 or the LIST-EXTENDED
// extension.
func (c *Client) List(ref, pattern string, options *ListOptions) *ListCommand {
	cmd := &ListCommand{
		mailboxes:    make(chan *ListData, 64),
		returnStatus: options != nil && len(options.ReturnStatus) > 0,
	}
	enc := c.beginCommand("LIST", cmd)
	if selectOpts := options.selectOpts(); len(selectOpts) > 0 {
		enc.SP().List(len(selectOpts), func(i int) {
			enc.Atom(selectOpts[i])
		})
	}
	enc.SP().Mailbox(ref).SP().String(pattern)
	if returnOpts := options.returnOpts(); len(returnOpts) > 0 {
		enc.SP().Atom("RETURN").SP().List(len(returnOpts), func(i int) {
			opt := returnOpts[i]
			enc.Atom(opt)
			if opt == "STATUS" {
				enc.SP().List(len(options.ReturnStatus), func(j int) {
					enc.Atom(string(options.ReturnStatus[j]))
				})
			}
		})
	}
	enc.end()
	return cmd
}

// ListCommand is a LIST command.
type ListCommand struct {
	cmd
	mailboxes chan *ListData

	returnStatus bool
	pendingData  *ListData
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

	// Extended data
	ChildInfo *ListDataChildInfo
	OldName   string
	Status    *StatusData
}

type ListDataChildInfo struct {
	Subscribed bool
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

	data.Delim, err = readDelim(dec)
	if err != nil {
		return nil, err
	}

	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	if data.Mailbox, err = dec.ExpectMailbox(); err != nil {
		return nil, err
	}

	if dec.SP() {
		err := dec.ExpectList(func() error {
			var tag string
			if !dec.ExpectAString(&tag) || !dec.ExpectSP() {
				return dec.Err()
			}
			var err error
			switch tag {
			case "CHILDINFO":
				data.ChildInfo, err = readChildInfoExtendedItem(dec)
				if err != nil {
					return fmt.Errorf("in childinfo-extended-item: %v", err)
				}
			case "OLDNAME":
				data.OldName, err = readOldNameExtendedItem(dec)
				if err != nil {
					return fmt.Errorf("in oldname-extended-item: %v", err)
				}
			default:
				if !dec.DiscardValue() {
					return fmt.Errorf("in tagged-ext-val: %v", err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("in mbox-list-extended: %v", err)
		}
	}

	return &data, nil
}

func readChildInfoExtendedItem(dec *imapwire.Decoder) (*ListDataChildInfo, error) {
	var childInfo ListDataChildInfo
	err := dec.ExpectList(func() error {
		var opt string
		if !dec.ExpectAString(&opt) {
			return dec.Err()
		}
		if opt == "SUBSCRIBED" {
			childInfo.Subscribed = true
		}
		return nil
	})
	return &childInfo, err
}

func readOldNameExtendedItem(dec *imapwire.Decoder) (string, error) {
	if !dec.ExpectSpecial('(') {
		return "", dec.Err()
	}
	name, err := dec.ExpectMailbox()
	if err != nil {
		return "", err
	}
	if !dec.ExpectSpecial(')') {
		return "", dec.Err()
	}
	return name, nil
}

func readDelim(dec *imapwire.Decoder) (rune, error) {
	var delimStr string
	if dec.Quoted(&delimStr) {
		delim, size := utf8.DecodeRuneInString(delimStr)
		if delim == utf8.RuneError || size != len(delimStr) {
			return 0, fmt.Errorf("mailbox delimiter must be a single rune")
		}
		return delim, nil
	} else if !dec.ExpectNIL() {
		return 0, dec.Err()
	} else {
		return 0, nil
	}
}
