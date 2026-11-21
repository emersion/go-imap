package imapserver

import (
	"bufio"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Helper to create a decoder from a command string (without tag and command name)
func newTestDecoder(s string) *imapwire.Decoder {
	br := bufio.NewReader(strings.NewReader(s))
	return imapwire.NewDecoder(br, imapwire.ConnSideServer)
}

func TestReadNotifyOptions_None(t *testing.T) {
	dec := newTestDecoder(" NONE\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options != nil {
		t.Errorf("Expected nil options for NOTIFY NONE, got %+v", options)
	}
}

func TestReadNotifyOptions_Selected(t *testing.T) {
	dec := newTestDecoder(" SET (SELECTED (MessageNew MessageExpunge))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	item := options.Items[0]
	if item.MailboxSpec != imap.NotifyMailboxSpecSelected {
		t.Errorf("Expected SELECTED, got %v", item.MailboxSpec)
	}
	if len(item.Events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(item.Events))
	}
	if item.Events[0] != imap.NotifyEventMessageNew {
		t.Errorf("Expected MessageNew, got %v", item.Events[0])
	}
	if item.Events[1] != imap.NotifyEventMessageExpunge {
		t.Errorf("Expected MessageExpunge, got %v", item.Events[1])
	}
}

func TestReadNotifyOptions_SelectedDelayed(t *testing.T) {
	dec := newTestDecoder(" SET (SELECTED-DELAYED (MessageNew MessageExpunge))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	if options.Items[0].MailboxSpec != imap.NotifyMailboxSpecSelectedDelayed {
		t.Errorf("Expected SELECTED-DELAYED, got %v", options.Items[0].MailboxSpec)
	}
}

func TestReadNotifyOptions_Personal(t *testing.T) {
	dec := newTestDecoder(" SET (PERSONAL (MailboxName SubscriptionChange))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	item := options.Items[0]
	if item.MailboxSpec != imap.NotifyMailboxSpecPersonal {
		t.Errorf("Expected PERSONAL, got %v", item.MailboxSpec)
	}
	if len(item.Events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(item.Events))
	}
}

func TestReadNotifyOptions_Inboxes(t *testing.T) {
	dec := newTestDecoder(" SET (INBOXES (MessageNew))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	if options.Items[0].MailboxSpec != imap.NotifyMailboxSpecInboxes {
		t.Errorf("Expected INBOXES, got %v", options.Items[0].MailboxSpec)
	}
}

func TestReadNotifyOptions_Subscribed(t *testing.T) {
	dec := newTestDecoder(" SET (SUBSCRIBED (MessageNew MailboxName))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	if options.Items[0].MailboxSpec != imap.NotifyMailboxSpecSubscribed {
		t.Errorf("Expected SUBSCRIBED, got %v", options.Items[0].MailboxSpec)
	}
}

func TestReadNotifyOptions_Subtree(t *testing.T) {
	dec := newTestDecoder(" SET (SUBTREE (INBOX Lists) (MessageNew))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	item := options.Items[0]
	if !item.Subtree {
		t.Error("Expected Subtree=true")
	}
	if len(item.Mailboxes) != 2 {
		t.Fatalf("Expected 2 mailboxes, got %d", len(item.Mailboxes))
	}
	if item.Mailboxes[0] != "INBOX" {
		t.Errorf("Expected INBOX, got %v", item.Mailboxes[0])
	}
	if item.Mailboxes[1] != "Lists" {
		t.Errorf("Expected Lists, got %v", item.Mailboxes[1])
	}
	if len(item.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(item.Events))
	}
	if item.Events[0] != imap.NotifyEventMessageNew {
		t.Errorf("Expected MessageNew, got %v", item.Events[0])
	}
}

func TestReadNotifyOptions_MailboxList(t *testing.T) {
	dec := newTestDecoder(" SET ((INBOX Sent) (MessageNew MessageExpunge FlagChange))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	item := options.Items[0]
	if item.Subtree {
		t.Error("Expected Subtree=false")
	}
	if len(item.Mailboxes) != 2 {
		t.Fatalf("Expected 2 mailboxes, got %d", len(item.Mailboxes))
	}
	if item.Mailboxes[0] != "INBOX" {
		t.Errorf("Expected INBOX, got %v", item.Mailboxes[0])
	}
	if item.Mailboxes[1] != "Sent" {
		t.Errorf("Expected Sent, got %v", item.Mailboxes[1])
	}
	if len(item.Events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(item.Events))
	}
}

func TestReadNotifyOptions_StatusIndicator(t *testing.T) {
	dec := newTestDecoder(" SET (STATUS) (SELECTED (MessageNew MessageExpunge))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if !options.Status {
		t.Error("Expected STATUS=true")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
}

func TestReadNotifyOptions_MultipleItems(t *testing.T) {
	dec := newTestDecoder(" SET (SELECTED (MessageNew MessageExpunge)) (PERSONAL (MailboxName SubscriptionChange)) (INBOXES (MessageNew))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(options.Items))
	}
	if options.Items[0].MailboxSpec != imap.NotifyMailboxSpecSelected {
		t.Errorf("Expected SELECTED, got %v", options.Items[0].MailboxSpec)
	}
	if options.Items[1].MailboxSpec != imap.NotifyMailboxSpecPersonal {
		t.Errorf("Expected PERSONAL, got %v", options.Items[1].MailboxSpec)
	}
	if options.Items[2].MailboxSpec != imap.NotifyMailboxSpecInboxes {
		t.Errorf("Expected INBOXES, got %v", options.Items[2].MailboxSpec)
	}
}

func TestReadNotifyOptions_AllEvents(t *testing.T) {
	dec := newTestDecoder(" SET (SELECTED (MessageNew MessageExpunge FlagChange AnnotationChange MailboxName SubscriptionChange MailboxMetadataChange ServerMetadataChange))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	if len(options.Items[0].Events) != 8 {
		t.Fatalf("Expected 8 events, got %d", len(options.Items[0].Events))
	}
	expectedEvents := []imap.NotifyEvent{
		imap.NotifyEventMessageNew,
		imap.NotifyEventMessageExpunge,
		imap.NotifyEventFlagChange,
		imap.NotifyEventAnnotationChange,
		imap.NotifyEventMailboxName,
		imap.NotifyEventSubscriptionChange,
		imap.NotifyEventMailboxMetadataChange,
		imap.NotifyEventServerMetadataChange,
	}
	for i, expected := range expectedEvents {
		if options.Items[0].Events[i] != expected {
			t.Errorf("Event %d: expected %v, got %v", i, expected, options.Items[0].Events[i])
		}
	}
}

func TestReadNotifyOptions_NoEventsSpecified(t *testing.T) {
	// Mailbox specifiers without event list should be valid
	dec := newTestDecoder(" SET (SELECTED)\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(options.Items))
	}
	if len(options.Items[0].Events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(options.Items[0].Events))
	}
}

// Error cases

func TestReadNotifyOptions_MissingSpace(t *testing.T) {
	dec := newTestDecoder("NONE\r\n")
	_, err := readNotifyOptions(dec)
	if err == nil {
		t.Fatal("Expected error for missing space after NOTIFY")
	}
}

func TestReadNotifyOptions_InvalidCommand(t *testing.T) {
	dec := newTestDecoder(" INVALID\r\n")
	_, err := readNotifyOptions(dec)
	if err == nil {
		t.Fatal("Expected error for invalid command")
	}
	imapErr, ok := err.(*imap.Error)
	if !ok {
		t.Fatalf("Expected *imap.Error, got %T", err)
	}
	if imapErr.Type != imap.StatusResponseTypeBad {
		t.Errorf("Expected BAD response, got %v", imapErr.Type)
	}
}

func TestReadNotifyOptions_SetWithoutItems(t *testing.T) {
	dec := newTestDecoder(" SET\r\n")
	_, err := readNotifyOptions(dec)
	if err == nil {
		t.Fatal("Expected error for SET without items")
	}
	imapErr, ok := err.(*imap.Error)
	if !ok {
		t.Fatalf("Expected *imap.Error, got %T", err)
	}
	if imapErr.Type != imap.StatusResponseTypeBad {
		t.Errorf("Expected BAD response, got %v", imapErr.Type)
	}
}

func TestReadNotifyOptions_SetWithEmptyList(t *testing.T) {
	dec := newTestDecoder(" SET ()\r\n")
	_, err := readNotifyOptions(dec)
	if err == nil {
		t.Fatal("Expected error for SET with empty list")
	}
}

func TestReadNotifyOptions_InvalidMailboxSpec(t *testing.T) {
	dec := newTestDecoder(" SET (INVALID (MessageNew))\r\n")
	_, err := readNotifyOptions(dec)
	if err == nil {
		t.Fatal("Expected error for invalid mailbox specifier")
	}
	imapErr, ok := err.(*imap.Error)
	if !ok {
		t.Fatalf("Expected *imap.Error, got %T", err)
	}
	if imapErr.Type != imap.StatusResponseTypeBad {
		t.Errorf("Expected BAD response, got %v", imapErr.Type)
	}
}

func TestReadNotifyOptions_InvalidEvent(t *testing.T) {
	dec := newTestDecoder(" SET (SELECTED (InvalidEvent))\r\n")
	_, err := readNotifyOptions(dec)
	if err == nil {
		t.Fatal("Expected error for invalid event")
	}
	imapErr, ok := err.(*imap.Error)
	if !ok {
		t.Fatalf("Expected *imap.Error, got %T", err)
	}
	if imapErr.Type != imap.StatusResponseTypeBad {
		t.Errorf("Expected BAD response, got %v", imapErr.Type)
	}
}

func TestReadNotifyOptions_SubtreeWithoutMailboxList(t *testing.T) {
	dec := newTestDecoder(" SET (SUBTREE)\r\n")
	_, err := readNotifyOptions(dec)
	if err == nil {
		t.Fatal("Expected error for SUBTREE without mailbox list")
	}
}

func TestReadNotifyOptions_MissingCRLF(t *testing.T) {
	dec := newTestDecoder(" SET (SELECTED (MessageNew))")
	_, err := readNotifyOptions(dec)
	if err == nil {
		t.Fatal("Expected error for missing CRLF")
	}
}

func TestReadNotifyOptions_StatusOnly(t *testing.T) {
	// STATUS alone is technically valid per the parser, though not very useful
	// The validation only requires at least one item when STATUS is false
	dec := newTestDecoder(" SET (STATUS)\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if !options.Status {
		t.Error("Expected STATUS=true")
	}
	if len(options.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(options.Items))
	}
}

func TestReadNotifyOptions_CaseInsensitive(t *testing.T) {
	// Test that commands and keywords are case-insensitive
	dec := newTestDecoder(" set (selected (MessageNew)) (personal (MailboxName))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if len(options.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(options.Items))
	}
}

func TestReadNotifyOptions_ComplexMixed(t *testing.T) {
	// Complex example with STATUS, multiple mailbox specs, and various events
	dec := newTestDecoder(" SET (STATUS) (SELECTED (MessageNew MessageExpunge)) (SUBTREE (INBOX) (MessageNew)) ((Drafts Sent) (FlagChange))\r\n")
	options, err := readNotifyOptions(dec)
	if err != nil {
		t.Fatalf("readNotifyOptions() error = %v", err)
	}
	if options == nil {
		t.Fatal("Expected non-nil options")
	}
	if !options.Status {
		t.Error("Expected STATUS=true")
	}
	if len(options.Items) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(options.Items))
	}

	// Check first item (SELECTED)
	if options.Items[0].MailboxSpec != imap.NotifyMailboxSpecSelected {
		t.Errorf("Item 0: Expected SELECTED, got %v", options.Items[0].MailboxSpec)
	}

	// Check second item (SUBTREE)
	if !options.Items[1].Subtree {
		t.Error("Item 1: Expected Subtree=true")
	}
	if len(options.Items[1].Mailboxes) != 1 || options.Items[1].Mailboxes[0] != "INBOX" {
		t.Errorf("Item 1: Expected mailboxes [INBOX], got %v", options.Items[1].Mailboxes)
	}

	// Check third item (mailbox list)
	if options.Items[2].Subtree {
		t.Error("Item 2: Expected Subtree=false")
	}
	if len(options.Items[2].Mailboxes) != 2 {
		t.Errorf("Item 2: Expected 2 mailboxes, got %d", len(options.Items[2].Mailboxes))
	}
}
