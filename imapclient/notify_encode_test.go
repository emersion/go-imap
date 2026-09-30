package imapclient

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func encodeToString(options *imap.NotifyOptions) (string, error) {
	buf := &bytes.Buffer{}
	bw := bufio.NewWriter(buf)
	enc := imapwire.NewEncoder(bw, imapwire.ConnSideClient)

	if err := encodeNotifyOptions(enc, options); err != nil {
		return "", err
	}

	enc.CRLF()
	bw.Flush()

	return buf.String(), nil
}

func TestEncodeNotifyOptions(t *testing.T) {
	tests := []struct {
		name     string
		options  *imap.NotifyOptions
		expected string
	}{
		{
			name:     "None",
			options:  nil,
			expected: " NONE\r\n",
		},
		{
			name: "EmptyItems",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{},
			},
			expected: " NONE\r\n",
		},
		{
			name: "Selected",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecSelected,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
							imap.NotifyEventMessageExpunge,
						},
					},
				},
			},
			expected: " SET (SELECTED (MessageNew MessageExpunge))\r\n",
		},
		{
			name: "SelectedDelayed",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecSelectedDelayed,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
							imap.NotifyEventMessageExpunge,
						},
					},
				},
			},
			expected: " SET (SELECTED-DELAYED (MessageNew MessageExpunge))\r\n",
		},
		{
			name: "Personal",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecPersonal,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMailboxName,
							imap.NotifyEventSubscriptionChange,
						},
					},
				},
			},
			expected: " SET (PERSONAL (MailboxName SubscriptionChange))\r\n",
		},
		{
			name: "Inboxes",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecInboxes,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
						},
					},
				},
			},
			expected: " SET (INBOXES (MessageNew))\r\n",
		},
		{
			name: "Subscribed",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecSubscribed,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
							imap.NotifyEventMailboxName,
						},
					},
				},
			},
			expected: " SET (SUBSCRIBED (MessageNew MailboxName))\r\n",
		},
		{
			name: "Subtree",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						Subtree:   true,
						Mailboxes: []string{"INBOX", "Lists"},
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
						},
					},
				},
			},
			expected: ` SET (SUBTREE (INBOX "Lists") (MessageNew))` + "\r\n",
		},
		{
			name: "MailboxList",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						Mailboxes: []string{"INBOX", "Sent"},
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
							imap.NotifyEventMessageExpunge,
							imap.NotifyEventFlagChange,
						},
					},
				},
			},
			expected: ` SET ((INBOX "Sent") (MessageNew MessageExpunge FlagChange))` + "\r\n",
		},
		{
			name: "StatusIndicator",
			options: &imap.NotifyOptions{
				Status: true,
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecSelected,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
							imap.NotifyEventMessageExpunge,
						},
					},
				},
			},
			expected: " SET (STATUS) (SELECTED (MessageNew MessageExpunge))\r\n",
		},
		{
			name: "MultipleItems",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecSelected,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
							imap.NotifyEventMessageExpunge,
						},
					},
					{
						MailboxSpec: imap.NotifyMailboxSpecPersonal,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMailboxName,
							imap.NotifyEventSubscriptionChange,
						},
					},
					{
						MailboxSpec: imap.NotifyMailboxSpecInboxes,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
						},
					},
				},
			},
			expected: " SET (SELECTED (MessageNew MessageExpunge)) (PERSONAL (MailboxName SubscriptionChange)) (INBOXES (MessageNew))\r\n",
		},
		{
			name: "AllEvents",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecSelected,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
							imap.NotifyEventMessageExpunge,
							imap.NotifyEventFlagChange,
							imap.NotifyEventAnnotationChange,
							imap.NotifyEventMailboxName,
							imap.NotifyEventSubscriptionChange,
							imap.NotifyEventMailboxMetadataChange,
							imap.NotifyEventServerMetadataChange,
						},
					},
				},
			},
			expected: " SET (SELECTED (MessageNew MessageExpunge FlagChange AnnotationChange MailboxName SubscriptionChange MailboxMetadataChange ServerMetadataChange))\r\n",
		},
		{
			name: "NoEvents",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecSelected,
						Events:      []imap.NotifyEvent{},
					},
				},
			},
			expected: " SET (SELECTED)\r\n",
		},
		{
			name: "ComplexMixed",
			options: &imap.NotifyOptions{
				Status: true,
				Items: []imap.NotifyItem{
					{
						MailboxSpec: imap.NotifyMailboxSpecSelected,
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
							imap.NotifyEventMessageExpunge,
						},
					},
					{
						Subtree:   true,
						Mailboxes: []string{"INBOX"},
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
						},
					},
					{
						Mailboxes: []string{"Drafts", "Sent"},
						Events: []imap.NotifyEvent{
							imap.NotifyEventFlagChange,
						},
					},
				},
			},
			expected: ` SET (STATUS) (SELECTED (MessageNew MessageExpunge)) (SUBTREE (INBOX) (MessageNew)) (("Drafts" "Sent") (FlagChange))` + "\r\n",
		},
		{
			name: "MailboxWithSpecialChars",
			options: &imap.NotifyOptions{
				Items: []imap.NotifyItem{
					{
						Mailboxes: []string{"INBOX", "Foo Bar", "Test&Mailbox"},
						Events: []imap.NotifyEvent{
							imap.NotifyEventMessageNew,
						},
					},
				},
			},
			expected: ` SET ((INBOX "Foo Bar" "Test&-Mailbox") (MessageNew))` + "\r\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := encodeToString(tc.options)
			if err != nil {
				t.Fatalf("encodeToString() error = %v", err)
			}
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestEncodeNotifyOptions_InvalidItem(t *testing.T) {
	// Items with neither MailboxSpec nor Mailboxes should return an error
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				// Invalid: no mailbox spec or mailboxes
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
				},
			},
		},
	}
	_, err := encodeToString(options)
	if err == nil {
		t.Fatal("Expected error for invalid NOTIFY item, got nil")
	}

	expectedMsg := "invalid NOTIFY item: must specify either MailboxSpec or Mailboxes"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}
