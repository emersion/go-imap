package imapclient

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func encodeToString(options *imap.NotifyOptions) string {
	buf := &bytes.Buffer{}
	bw := bufio.NewWriter(buf)
	enc := imapwire.NewEncoder(bw, imapwire.ConnSideClient)

	encodeNotifyOptions(enc, options)

	enc.CRLF()
	bw.Flush()

	return buf.String()
}

func TestEncodeNotifyOptions_None(t *testing.T) {
	result := encodeToString(nil)
	expected := " NONE\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_EmptyItems(t *testing.T) {
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{},
	}
	result := encodeToString(options)
	expected := " NONE\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_Selected(t *testing.T) {
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				MailboxSpec: imap.NotifyMailboxSpecSelected,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
					imap.NotifyEventMessageExpunge,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (SELECTED (MessageNew MessageExpunge))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_SelectedDelayed(t *testing.T) {
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				MailboxSpec: imap.NotifyMailboxSpecSelectedDelayed,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
					imap.NotifyEventMessageExpunge,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (SELECTED-DELAYED (MessageNew MessageExpunge))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_Personal(t *testing.T) {
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				MailboxSpec: imap.NotifyMailboxSpecPersonal,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMailboxName,
					imap.NotifyEventSubscriptionChange,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (PERSONAL (MailboxName SubscriptionChange))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_Inboxes(t *testing.T) {
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				MailboxSpec: imap.NotifyMailboxSpecInboxes,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (INBOXES (MessageNew))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_Subscribed(t *testing.T) {
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				MailboxSpec: imap.NotifyMailboxSpecSubscribed,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
					imap.NotifyEventMailboxName,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (SUBSCRIBED (MessageNew MailboxName))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_Subtree(t *testing.T) {
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				Subtree:   true,
				Mailboxes: []string{"INBOX", "Lists"},
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (SUBTREE (INBOX \"Lists\") (MessageNew))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_MailboxList(t *testing.T) {
	options := &imap.NotifyOptions{
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
	}
	result := encodeToString(options)
	expected := " SET ((INBOX \"Sent\") (MessageNew MessageExpunge FlagChange))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_StatusIndicator(t *testing.T) {
	options := &imap.NotifyOptions{
		STATUS: true,
		Items: []imap.NotifyItem{
			{
				MailboxSpec: imap.NotifyMailboxSpecSelected,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
					imap.NotifyEventMessageExpunge,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (STATUS) (SELECTED (MessageNew MessageExpunge))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_MultipleItems(t *testing.T) {
	options := &imap.NotifyOptions{
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
	}
	result := encodeToString(options)
	expected := " SET (SELECTED (MessageNew MessageExpunge)) (PERSONAL (MailboxName SubscriptionChange)) (INBOXES (MessageNew))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_AllEvents(t *testing.T) {
	options := &imap.NotifyOptions{
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
	}
	result := encodeToString(options)
	expected := " SET (SELECTED (MessageNew MessageExpunge FlagChange AnnotationChange MailboxName SubscriptionChange MailboxMetadataChange ServerMetadataChange))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_NoEvents(t *testing.T) {
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				MailboxSpec: imap.NotifyMailboxSpecSelected,
				Events:      []imap.NotifyEvent{},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (SELECTED)\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_InvalidItemSkipped(t *testing.T) {
	// Items with neither MailboxSpec nor Mailboxes should be skipped
	// XXX: should we warn about these?
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				// Invalid: no mailbox spec or mailboxes
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
				},
			},
			{
				MailboxSpec: imap.NotifyMailboxSpecSelected,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET (SELECTED (MessageNew))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_ComplexMixed(t *testing.T) {
	options := &imap.NotifyOptions{
		STATUS: true,
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
	}
	result := encodeToString(options)
	expected := " SET (STATUS) (SELECTED (MessageNew MessageExpunge)) (SUBTREE (INBOX) (MessageNew)) ((\"Drafts\" \"Sent\") (FlagChange))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestEncodeNotifyOptions_MailboxWithSpecialChars(t *testing.T) {
	// Test mailbox names that require quoting
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				Mailboxes: []string{"INBOX", "Foo Bar", "Test&Mailbox"},
				Events: []imap.NotifyEvent{
					imap.NotifyEventMessageNew,
				},
			},
		},
	}
	result := encodeToString(options)
	expected := " SET ((INBOX \"Foo Bar\" \"Test&-Mailbox\") (MessageNew))\r\n"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
