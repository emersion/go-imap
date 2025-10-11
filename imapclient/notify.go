package imapclient

import (
	"sync/atomic"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Notify sends a NOTIFY command (RFC 5465).
//
// The NOTIFY command allows clients to request server-push notifications
// for mailbox events like new messages, expunges, flag changes, etc.
//
// When NOTIFY SET is active, the server may send unsolicited responses at any
// time (STATUS, FETCH, EXPUNGE, LIST responses). These unsolicited responses
// are delivered via the UnilateralDataHandler callbacks set in
// imapclient.Options.
//
// When the server sends an untagged OK [NOTIFICATIONOVERFLOW] response, the
// Overflow() channel on the returned NotifyCommand will be closed. This
// indicates the server has disabled all notifications and the client should
// re-issue the NOTIFY command if needed.
//
// This requires support for the NOTIFY extension.
func (c *Client) Notify(options *imap.NotifyOptions) (*NotifyCommand, error) {
	cmd := &NotifyCommand{
		options:  options,
		overflow: make(chan struct{}),
	}
	enc := c.beginCommand("NOTIFY", cmd)
	encodeNotifyOptions(enc.Encoder, options)
	enc.end()

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// encodeNotifyOptions encodes NOTIFY command options to the encoder.
func encodeNotifyOptions(enc *imapwire.Encoder, options *imap.NotifyOptions) {
	if options == nil || len(options.Items) == 0 {
		// NOTIFY NONE - disable all notifications
		enc.SP().Atom("NONE")
	} else {
		// NOTIFY SET
		enc.SP().Atom("SET")

		if options.STATUS {
			enc.SP().List(1, func(i int) {
				enc.Atom("STATUS")
			})
		}

		// Encode each notify item
		for _, item := range options.Items {
			// Validate the item before encoding
			if item.MailboxSpec == "" && len(item.Mailboxes) == 0 {
				// Skip invalid items - this shouldn't happen with properly constructed NotifyOptions
				continue
			}

			enc.SP().List(1, func(i int) {
				// Encode mailbox specification
				if item.MailboxSpec != "" {
					enc.Atom(string(item.MailboxSpec))
				} else if len(item.Mailboxes) > 0 {
					if item.Subtree {
						enc.Atom("SUBTREE").SP()
					}
					// Encode mailbox list
					enc.List(len(item.Mailboxes), func(j int) {
						enc.Mailbox(item.Mailboxes[j])
					})
				}

				// Encode events
				if len(item.Events) > 0 {
					enc.SP().List(len(item.Events), func(j int) {
						enc.Atom(string(item.Events[j]))
					})
				}
			})
		}
	}
}

// NotifyNone sends a NOTIFY NONE command to disable all notifications.
func (c *Client) NotifyNone() error {
	_, err := c.Notify(nil)
	return err
}

// NotifyCommand is a NOTIFY command.
//
// When NOTIFY SET is active (options != nil), the server may send unsolicited
// responses at any time. These responses are delivered via UnilateralDataHandler
// (see Options.UnilateralDataHandler).
//
// The Overflow() channel can be monitored to detect when the server sends an
// untagged OK [NOTIFICATIONOVERFLOW] response, indicating that notifications
// shall no longer be delivered.
type NotifyCommand struct {
	commandBase

	options  *imap.NotifyOptions
	overflow chan struct{}
	closed   atomic.Bool
}

// Wait blocks until the NOTIFY command has completed.
func (cmd *NotifyCommand) Wait() error {
	return cmd.wait()
}

// Overflow returns a channel that is closed when the server sends a
// NOTIFICATIONOVERFLOW response code. This indicates the server has disabled
// notifications and the client should re-issue the NOTIFY command if needed.
//
// The channel is nil if NOTIFY NONE was sent (no notifications active).
func (cmd *NotifyCommand) Overflow() <-chan struct{} {
	if cmd.options == nil || len(cmd.options.Items) == 0 {
		return nil
	}
	return cmd.overflow
}

// Close disables the NOTIFY monitoring by calling it an internal close.
// This is called internally when NOTIFICATIONOVERFLOW is received.
func (cmd *NotifyCommand) close() {
	if cmd.closed.Swap(true) {
		return
	}
	close(cmd.overflow)
}

func (cmd *NotifyCommand) handleOverflow() {
	cmd.close()
}
