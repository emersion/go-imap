package imapclient_test

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestSearch(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	criteria := imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{{
			Key:   "Message-Id",
			Value: "<191101702316132@example.com>",
		}},
	}
	data, err := client.Search(&criteria, nil).Wait()
	if err != nil {
		t.Fatalf("Search().Wait() = %v", err)
	}
	seqSet, ok := data.All.(imap.SeqSet)
	if !ok {
		t.Fatalf("SearchData.All = %T, want SeqSet", data.All)
	}
	nums, _ := seqSet.Nums()
	want := []uint32{1}
	if !reflect.DeepEqual(nums, want) {
		t.Errorf("SearchData.All.Nums() = %v, want %v", nums, want)
	}
}

func TestSearchCondStore(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapCondStore) {
		t.Skip("CONDSTORE not supported")
	}

	// SELECT directly to capture HIGHESTMODSEQ.
	selectData, err := client.Select("INBOX", &imap.SelectOptions{
		CondStore: true,
	}).Wait()
	if err != nil {
		t.Fatalf("Select().Wait() = %v", err)
	}
	highestModSeq := selectData.HighestModSeq
	if highestModSeq == 0 {
		t.Fatalf("highestModSeq = %v, no CONDSTORE support", highestModSeq)
	}

	criteria := imap.SearchCriteria{
		ModSeq: &imap.SearchCriteriaModSeq{
			ModSeq: highestModSeq, // Unlike FETCH, it is inclusive.
		},
	}
	seqSet := imap.SeqSetNum(1)

	// Do STORE to advance MODSEQ.
	storeData, err := client.Store(seqSet, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagAnswered},
	}, &imap.StoreOptions{
		UnchangedSince: highestModSeq,
	}).Collect()
	if err != nil {
		t.Fatalf("failed to store: %v", err)
	}
	if len(storeData) != 1 {
		t.Fatalf("len(storeData) = %v, want 1", len(storeData))
	}
	updatedModSeq := storeData[0].ModSeq
	if updatedModSeq <= highestModSeq {
		t.Errorf("storeData[0].ModSeq = %v, want a larger value", updatedModSeq)
	}

	// MODSEQ previous-HIGHESTMODSEQ will return changed message.
	data, err := client.Search(&criteria, nil).Wait()
	if err != nil {
		t.Fatalf("Search().Wait() = %v", err)
	}
	seqSet, ok := data.All.(imap.SeqSet)
	if !ok {
		t.Fatalf("SearchData.All = %T, want SeqSet", data.All)
	}
	nums, _ := seqSet.Nums()
	want := []uint32{1}
	if !reflect.DeepEqual(nums, want) {
		t.Errorf("SearchData.All.Nums() = %v, want %v", nums, want)
	}
}

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

func TestESearchCondStore(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapESearch) {
		t.Skip("ESEARCH not supported")
	}
	if !client.Caps().Has(imap.CapCondStore) {
		t.Skip("CONDSTORE not supported")
	}

	// Mostly similar to TestSearchCondstore but uses ReturnCount.

	// SELECT directly to capture HIGHESTMODSEQ.
	selectData, err := client.Select("INBOX", &imap.SelectOptions{
		CondStore: true,
	}).Wait()
	if err != nil {
		t.Fatalf("Select().Wait() = %v", err)
	}
	highestModSeq := selectData.HighestModSeq
	if highestModSeq == 0 {
		t.Fatalf("highestModSeq = %v, no CONDSTORE support", highestModSeq)
	}

	seqSet := imap.SeqSetNum(1)
	criteria := imap.SearchCriteria{
		ModSeq: &imap.SearchCriteriaModSeq{
			ModSeq: highestModSeq,
		},
	}
	opts := imap.SearchOptions{
		ReturnCount: true,
	}

	// Do STORE to advance MODSEQ.
	// CONDSTORE-enabled connection (enabled by MODSEQ search) will always return FETCH with updated MODSEQ.
	storeData, err := client.Store(seqSet, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagAnswered},
	}, &imap.StoreOptions{
		UnchangedSince: highestModSeq,
	}).Collect()
	if err != nil {
		t.Fatalf("failed to store: %v", err)
	}
	if len(storeData) != 1 {
		t.Fatalf("len(storeData) = %v, want 1", len(storeData))
	}
	updatedModSeq := storeData[0].ModSeq
	if updatedModSeq <= highestModSeq {
		t.Errorf("storeData[0].ModSeq = %v, want a larger value", updatedModSeq)
	}

	// MODSEQ previous-HIGHESTMODSEQ will return changed message.
	data, err := client.Search(&criteria, &opts).Wait()
	if err != nil {
		t.Fatalf("Search().Wait() = %v", err)
	}
	if want := uint32(1); data.Count != want {
		t.Errorf("Count = %v, want %v", data.Count, want)
	}
}
