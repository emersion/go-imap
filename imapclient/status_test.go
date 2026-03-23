package imapclient_test

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestStatus(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	options := imap.StatusOptions{
		NumMessages: true,
		NumUnseen:   true,
	}
	data, err := client.Status("INBOX", &options).Wait()
	if err != nil {
		t.Fatalf("Status() = %v", err)
	}

	wantNumMessages := uint32(1)
	wantNumUnseen := uint32(1)
	want := &imap.StatusData{
		Mailbox:     "INBOX",
		NumMessages: &wantNumMessages,
		NumUnseen:   &wantNumUnseen,
	}
	if !reflect.DeepEqual(data, want) {
		t.Errorf("Status() = %#v but want %#v", data, want)
	}
}
