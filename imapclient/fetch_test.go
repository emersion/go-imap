package imapclient_test

import (
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestFetch(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	seqSet := imap.SeqSetNum(1)
	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}
	messages, err := client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("failed to fetch first message: %v", err)
	} else if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}

	msg := messages[0]
	if len(msg.BodySection) != 1 {
		t.Fatalf("len(msg.BodySection) = %v, want 1", len(msg.BodySection))
	}
	b := msg.FindBodySection(bodySection)
	if b == nil {
		t.Fatalf("FindBodySection() = nil")
	}
	body := strings.ReplaceAll(string(b), "\r\n", "\n")
	if body != simpleRawMessage {
		t.Errorf("body mismatch: got \n%v\n but want \n%v", body, simpleRawMessage)
	}
}

func TestFetchChangedSince(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapCondStore) {
		t.Skip("CONDSTORE not supported")
	}

	// SELECT manually to get HIGHESTMODSEQ.
	selectData, err := client.Select("INBOX", &imap.SelectOptions{
		CondStore: true,
	}).Wait()
	if err != nil {
		t.Fatalf("failed to select data: %v", err)
	}
	highestModSeq := selectData.HighestModSeq
	if highestModSeq == 0 {
		t.Fatalf("highestModSeq = %v, no CONDSTORE support", highestModSeq)
	}

	// CHANGEDSINCE=HIGHESTMODSEQ will yield empty FETCH.
	seqSet := imap.SeqSetNum(1)
	fetchOptions := &imap.FetchOptions{
		UID:          true,
		ChangedSince: highestModSeq,
	}
	messages, err := client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("len(messages) = %v, want 0", len(messages))
	}

	// Do STORE to advance MODSEQ.
	_, err = client.Store(seqSet, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagAnswered},
	}, nil).Collect()
	if err != nil {
		t.Fatalf("failed to store: %v", err)
	}

	// Now CHANGEDSINCE will include changed message.
	messages, err = client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("failed to fetch: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
}

func TestFetchPartialOffsetEcho(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	const offset = int64(1)<<32 + 5

	bodySection := &imap.FetchItemBodySection{
		Partial: &imap.SectionPartial{Offset: offset, Size: 10},
	}
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(messages) != 1 || len(messages[0].BodySection) != 1 {
		t.Fatalf("got %v messages", len(messages))
	}

	got := messages[0].BodySection[0].Section.Partial
	if got == nil || got.Offset != offset {
		t.Errorf("echoed partial offset = %v, want %v", got, offset)
	}
	if messages[0].FindBodySection(bodySection) == nil {
		t.Errorf("FindBodySection() = nil, want the section the client asked for")
	}
}
