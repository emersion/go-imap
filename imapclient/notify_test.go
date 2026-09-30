package imapclient_test

import (
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func TestClient_Notify(t *testing.T) {
	existsCh := make(chan uint32, 1)

	options := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Expunge: func(seqNum uint32) {
				// Not testing expunge in this test
			},
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					select {
					case existsCh <- *data.NumMessages:
					default:
					}
				}
			},
		},
	}

	client, server := newClientServerPairWithOptions(t, imap.ConnStateSelected, options)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapNotify) {
		t.Skip("NOTIFY not supported")
	}

	selectData, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() = %v", err)
	}
	initialExists := selectData.NumMessages

	notifyOptions := &imap.NotifyOptions{
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
	cmd, err := client.Notify(notifyOptions)
	if err != nil {
		t.Fatalf("Notify() = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}

	// Append a new message to INBOX (we should get a NOTIFY event for it).
	testMessage := `From: sender@example.com
To: recipient@example.com
Subject: Test NOTIFY

This is a test message for NOTIFY.
`
	appendCmd := client.Append("INBOX", int64(len(testMessage)), nil)
	appendCmd.Write([]byte(testMessage))
	appendCmd.Close()
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("Append() = %v", err)
	}

	// Wait for the EXISTS notification (with timeout)
	select {
	case count := <-existsCh:
		if count <= initialExists {
			t.Errorf("Expected EXISTS count > %d, got %d", initialExists, count)
		}
		t.Logf("Received EXISTS notification: %d messages (was %d)", count, initialExists)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for EXISTS notification")
	}
}

func TestClient_NotifyNone(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapNotify) {
		t.Skip("NOTIFY not supported")
	}

	cmd, err := client.Notify(nil)
	if err != nil {
		t.Fatalf("NotifyNone() = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}
}

func TestClient_NotifyMultiple(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapNotify) {
		t.Skip("NOTIFY not supported")
	}

	// Test NOTIFY with multiple items
	// Note: Dovecot doesn't support STATUS with message events
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
		},
	}

	cmd, err := client.Notify(options)
	if err != nil {
		t.Fatalf("Notify() = %v", err)
	}

	if err = cmd.Wait(); err != nil {
		t.Fatalf("Notify.Wait() = %v", err)
	}
}

// TestClient_NotifyPersonalMailboxes tests NOTIFY for personal mailboxes
func TestClient_NotifyPersonalMailboxes(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapNotify) {
		t.Skip("NOTIFY not supported")
	}

	// Note: Dovecot doesn't support message events with PERSONAL spec
	// Only mailbox events (MailboxName, SubscriptionChange) seem to work.
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

	cmd, err := client.Notify(options)
	if err != nil {
		t.Fatalf("Notify() = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}
}

// TestClient_NotifySubtree tests NOTIFY for mailbox subtrees
func TestClient_NotifySubtree(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapNotify) {
		t.Skip("NOTIFY not supported")
	}

	// Request notifications for INBOX subtree
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				Mailboxes: []string{"INBOX"},
				Subtree:   true,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMailboxName,
					imap.NotifyEventSubscriptionChange,
				},
			},
		},
	}

	cmd, err := client.Notify(options)
	if err != nil {
		t.Fatalf("Notify() = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}
}

// TestClient_NotifyMailboxes tests NOTIFY for specific mailboxes with message events
func TestClient_NotifyMailboxes(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapNotify) {
		t.Skip("NOTIFY not supported")
	}

	// Request notifications for specific mailboxes with SUBTREE
	// Note: Dovecot requires SUBTREE for explicit mailbox specifications
	options := &imap.NotifyOptions{
		Items: []imap.NotifyItem{
			{
				Mailboxes: []string{"INBOX"},
				Subtree:   true,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMailboxName,
					imap.NotifyEventSubscriptionChange,
				},
			},
		},
	}

	cmd, err := client.Notify(options)
	if err != nil {
		t.Fatalf("Notify() = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}
}

// TestClient_NotifySelectedDelayed tests NOTIFY with SELECTED-DELAYED for safe MSN usage
func TestClient_NotifySelectedDelayed(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapNotify) {
		t.Skip("NOTIFY not supported")
	}

	// Request notifications with SELECTED-DELAYED to defer expunge notifications
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

	cmd, err := client.Notify(options)
	if err != nil {
		t.Fatalf("Notify() = %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}
}

// TestClient_NotifySequence tests a sequence of NOTIFY commands
func TestClient_NotifySequence(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapNotify) {
		t.Skip("NOTIFY not supported")
	}

	// First NOTIFY command
	options1 := &imap.NotifyOptions{
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

	cmd1, err := client.Notify(options1)
	if err != nil {
		t.Fatalf("First Notify() = %v", err)
	}

	if err := cmd1.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}

	// Replace with different NOTIFY settings
	options2 := &imap.NotifyOptions{
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

	cmd2, err := client.Notify(options2)
	if err != nil {
		t.Fatalf("Second Notify() = %v", err)
	}
	if err := cmd2.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}

	// Disable all notifications
	cmd3, err := client.Notify(nil)
	if err != nil {
		t.Fatalf("NotifyNone() = %v", err)
	}

	if err := cmd3.Wait(); err != nil {
		t.Fatal("NotifyCommand.Wait failed")
	}
}
