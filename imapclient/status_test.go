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

func TestStatusCondStore(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapCondStore) {
		t.Skip("CONDSTORE not supported")
	}

	options := imap.StatusOptions{
		HighestModSeq: true,
	}
	data, err := client.Status("INBOX", &options).Wait()
	if err != nil {
		t.Fatalf("Status() = %v", err)
	}

	// Do STORE to advance HIGHESTMODSEQ.
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	msgs, err := client.Store(imap.SeqSetNum(1), &storeFlags, &imap.StoreOptions{
		UnchangedSince: data.HighestModSeq,
	}).Collect()
	if err != nil {
		t.Fatalf("Store().Collect() = %v", err)
	} else if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %v, want %v", len(msgs), 1)
	}

	if err := client.Unselect().Wait(); err != nil {
		t.Fatalf("Unselect() = %v", err)
	}

	data2, err := client.Status("INBOX", &options).Wait()
	if err != nil {
		t.Fatalf("Status() = %v", err)
	}

	if data2.HighestModSeq <= data.HighestModSeq {
		t.Error("HIGHESTMODSEQ should increase after STORE")
	}
}
