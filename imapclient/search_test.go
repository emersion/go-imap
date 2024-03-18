package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestESearch(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapESearch) {
		t.Skip("server doesn't support ESEARCH")
	}

	criteria := imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{{
			Key:   "Message-Id",
			Value: "<191101702316132@example.com>",
		}},
	}
	options := imap.SearchOptions{
		ReturnCount: true,
	}
	data, err := client.Search(&criteria, &options).Wait()
	if err != nil {
		t.Fatalf("Search().Wait() = %v", err)
	}
	if want := uint32(1); data.Count != want {
		t.Errorf("Count = %v, want %v", data.Count, want)
	}
}
