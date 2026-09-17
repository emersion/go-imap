package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleNotify(dec *imapwire.Decoder) error {
	options, err := readNotifyOptions(dec)
	if err != nil {
		return err
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	session, ok := c.session.(SessionNotify)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "NOTIFY not supported",
		}
	}

	w := &UpdateWriter{conn: c, allowExpunge: true}
	return session.Notify(w, options)
}

// readNotifyOptions parses the NOTIFY command arguments from the decoder.
// Returns nil options for NOTIFY NONE, or populated options for NOTIFY SET.
func readNotifyOptions(dec *imapwire.Decoder) (*imap.NotifyOptions, error) {
	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	// Check for NONE or SET
	var atom string
	if !dec.ExpectAtom(&atom) {
		return nil, dec.Err()
	}

	atom = strings.ToUpper(atom)
	if atom == "NONE" {
		// NOTIFY NONE - disable all notifications
		if !dec.ExpectCRLF() {
			return nil, dec.Err()
		}
		return nil, nil
	} else if atom == "SET" {
		// NOTIFY SET - set notifications
		options := &imap.NotifyOptions{}

		// Parse items until we hit CRLF
		for {
			// We need at least a space before each item
			if !dec.SP() {
				return nil, &imap.Error{
					Type: imap.StatusResponseTypeBad,
					Text: "Expected SP after SET or between items",
				}
			}

			// Parse a list
			isList, err := dec.List(func() error {
				// First element in the list: check if it's STATUS or a mailbox spec
				var firstAtom string
				if dec.Atom(&firstAtom) {
					firstAtom = strings.ToUpper(firstAtom)
					if firstAtom == "STATUS" {
						// This is the STATUS parameter
						options.Status = true
						return nil
					}

					// It's a mailbox spec or SUBTREE, parse as a notify item
					item, err := parseNotifyItemFromAtom(dec, firstAtom)
					if err != nil {
						return err
					}
					options.Items = append(options.Items, *item)
					return nil
				}

				// Not an atom, try to parse as mailbox list
				item, err := parseNotifyItemMailboxList(dec)
				if err != nil {
					return err
				}
				options.Items = append(options.Items, *item)
				return nil
			})
			if err != nil {
				return nil, err
			}
			if !isList {
				return nil, &imap.Error{
					Type: imap.StatusResponseTypeBad,
					Text: "Expected list",
				}
			}

			// Check if we're done (CRLF)
			if dec.CRLF() {
				break
			}
		}

		if len(options.Items) == 0 && !options.Status {
			return nil, &imap.Error{
				Type: imap.StatusResponseTypeBad,
				Text: "NOTIFY SET requires at least one mailbox specification",
			}
		}

		return options, nil
	} else {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Expected NONE or SET",
		}
	}
}

// parseNotifyItemFromAtom parses a notify item that starts with an atom (mailbox spec or SUBTREE)
func parseNotifyItemFromAtom(dec *imapwire.Decoder, firstAtom string) (*imap.NotifyItem, error) {
	item := &imap.NotifyItem{}

	switch firstAtom {
	case "SELECTED", "SELECTED-DELAYED", "PERSONAL", "INBOXES", "SUBSCRIBED":
		item.MailboxSpec = imap.NotifyMailboxSpec(firstAtom)

		// Check for optional events list
		if dec.SP() {
			err := dec.ExpectList(func() error {
				return readNotifyEvent(dec, item)
			})
			if err != nil {
				return nil, err
			}
		}
		return item, nil

	case "SUBTREE":
		// SUBTREE mailbox-list [event-list]
		item.Subtree = true

		if !dec.ExpectSP() {
			return nil, dec.Err()
		}

		// Read mailbox list
		err := dec.ExpectList(func() error {
			var mailbox string
			if !dec.ExpectMailbox(&mailbox) {
				return dec.Err()
			}
			item.Mailboxes = append(item.Mailboxes, mailbox)
			return nil
		})
		if err != nil {
			return nil, err
		}

		// Check for optional events list
		if dec.SP() {
			err := dec.ExpectList(func() error {
				return readNotifyEvent(dec, item)
			})
			if err != nil {
				return nil, err
			}
		}
		return item, nil

	default:
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Invalid mailbox specification: " + firstAtom,
		}
	}
}

// parseNotifyItemMailboxList parses a notify item that starts with a mailbox list
func parseNotifyItemMailboxList(dec *imapwire.Decoder) (*imap.NotifyItem, error) {
	item := &imap.NotifyItem{}

	// We're already inside a list, so we need to see if the first element is a mailbox or a list
	// The decoder is positioned at the start of the list content
	// Try to parse as a nested mailbox list
	isList, err := dec.List(func() error {
		var mailbox string
		if !dec.ExpectMailbox(&mailbox) {
			return dec.Err()
		}
		item.Mailboxes = append(item.Mailboxes, mailbox)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !isList {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Expected mailbox list",
		}
	}

	// Check for optional events list
	if dec.SP() {
		err := dec.ExpectList(func() error {
			return readNotifyEvent(dec, item)
		})
		if err != nil {
			return nil, err
		}
	}

	return item, nil
}

func readNotifyEvent(dec *imapwire.Decoder, item *imap.NotifyItem) error {
	var event string
	if !dec.ExpectAtom(&event) {
		return dec.Err()
	}

	// Validate event name
	switch imap.NotifyEvent(event) {
	case imap.NotifyEventMessageNew,
		imap.NotifyEventMessageExpunge,
		imap.NotifyEventFlagChange,
		imap.NotifyEventAnnotationChange,
		imap.NotifyEventMailboxName,
		imap.NotifyEventSubscriptionChange,
		imap.NotifyEventMailboxMetadataChange,
		imap.NotifyEventServerMetadataChange:
		item.Events = append(item.Events, imap.NotifyEvent(event))
		return nil
	default:
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown NOTIFY event: " + event,
		}
	}
}
