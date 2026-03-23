package imapclient_test

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestList(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	options := imap.ListOptions{
		ReturnStatus: &imap.StatusOptions{
			NumMessages: true,
		},
	}
	mailboxes, err := client.List("", "%", &options).Collect()
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	if len(mailboxes) != 1 {
		t.Fatalf("List() returned %v mailboxes, want 1", len(mailboxes))
	}
	mbox := mailboxes[0]

	wantNumMessages := uint32(1)
	want := &imap.ListData{
		Delim:   '/',
		Mailbox: "INBOX",
		Status: &imap.StatusData{
			Mailbox:     "INBOX",
			NumMessages: &wantNumMessages,
		},
	}
	if !reflect.DeepEqual(mbox, want) {
		t.Errorf("got %#v but want %#v", mbox, want)
	}
}
