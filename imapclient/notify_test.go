package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestClient_Notify(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Test NOTIFY with SELECTED mailbox
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

	// Note: The test server's stub implementation refuses NOTIFY with
	// NO [NOTIFICATIONOVERFLOW], which is RFC-compliant per RFC 5465 Section 3.1.
	_, err := client.Notify(options)
	if err == nil {
		t.Fatal("Expected error from stub implementation")
	}

	imapErr, ok := err.(*imap.Error)
	if !ok {
		t.Fatalf("Expected *imap.Error, got %T", err)
	}
	if imapErr.Type != imap.StatusResponseTypeNo {
		t.Errorf("Expected NO response, got %v", imapErr.Type)
	}
	if imapErr.Code != imap.ResponseCodeNotificationOverflow {
		t.Errorf("Expected NOTIFICATIONOVERFLOW code, got %v", imapErr.Code)
	}
}

func TestClient_NotifyNone(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Note: The test server's stub implementation refuses NOTIFY with
	// NO [NOTIFICATIONOVERFLOW]
	err := client.NotifyNone()
	if err == nil {
		t.Fatal("Expected error from stub implementation")
	}
	imapErr, ok := err.(*imap.Error)
	if !ok {
		t.Fatalf("Expected *imap.Error, got %T", err)
	}
	if imapErr.Type != imap.StatusResponseTypeNo {
		t.Errorf("Expected NO response, got %v", imapErr.Type)
	}
	if imapErr.Code != imap.ResponseCodeNotificationOverflow {
		t.Errorf("Expected NOTIFICATIONOVERFLOW code, got %v", imapErr.Code)
	}
}

func TestClient_NotifyMultiple(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Test NOTIFY with multiple items
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
				MailboxSpec: imap.NotifyMailboxSpecPersonal,
				Events: []imap.NotifyEvent{
					imap.NotifyEventMailboxName,
					imap.NotifyEventSubscriptionChange,
				},
			},
		},
	}

	// Note: The test server's stub implementation refuses NOTIFY with
	// NO [NOTIFICATIONOVERFLOW]
	_, err := client.Notify(options)
	if err == nil {
		t.Fatal("Expected error from stub implementation")
	}

	imapErr, ok := err.(*imap.Error)
	if !ok {
		t.Fatalf("Expected *imap.Error, got %T", err)
	}
	if imapErr.Type != imap.StatusResponseTypeNo {
		t.Errorf("Expected NO response, got %v", imapErr.Type)
	}
	if imapErr.Code != imap.ResponseCodeNotificationOverflow {
		t.Errorf("Expected NOTIFICATIONOVERFLOW code, got %v", imapErr.Code)
	}
}
