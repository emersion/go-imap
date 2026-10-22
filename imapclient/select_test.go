package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestSelect(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	data, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() = %v", err)
	} else if data.NumMessages != 1 {
		t.Errorf("SelectData.NumMessages = %v, want %v", data.NumMessages, 1)
	} else if data.ReadOnly {
		t.Errorf("SelectData.ReadOnly = true, want false")
	}
}

func TestSelectReadOnly(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	data, err := client.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		t.Fatalf("Select() = %v", err)
	} else if !data.ReadOnly {
		t.Errorf("SelectData.ReadOnly = false, want true")
	}
}
