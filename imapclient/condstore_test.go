package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestSelect_CondStore(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Test SELECT with CONDSTORE parameter
	options := &imap.SelectOptions{
		CondStore: true,
	}
	data, err := client.Select("INBOX", options).Wait()
	if err != nil {
		t.Fatalf("Select() with CONDSTORE = %v", err)
	}

	// Verify that HighestModSeq is returned
	if data.HighestModSeq == 0 {
		t.Errorf("SelectData.HighestModSeq is 0, expected non-zero value when CONDSTORE is enabled")
	}
	t.Logf("Mailbox HIGHESTMODSEQ: %d", data.HighestModSeq)
}

func TestFetch_ModSeq(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	// Test FETCH with MODSEQ item
	seqSet := imap.SeqSetNum(1)
	fetchOptions := &imap.FetchOptions{
		ModSeq: true,
	}
	messages, err := client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("Fetch() with MODSEQ = %v", err)
	} else if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}

	msg := messages[0]
	if msg.ModSeq == 0 {
		t.Errorf("msg.ModSeq is 0, expected non-zero value")
	}
	t.Logf("Message MODSEQ: %d", msg.ModSeq)
}

func TestFetch_ChangedSince(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	// First, get current ModSeq
	seqSet := imap.SeqSetNum(1)
	firstFetch, err := client.Fetch(seqSet, &imap.FetchOptions{
		ModSeq: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Initial Fetch() = %v", err)
	}
	currentModSeq := firstFetch[0].ModSeq
	t.Logf("Initial ModSeq: %d", currentModSeq)

	// Now fetch with CHANGEDSINCE using the current ModSeq
	// This should return no messages since nothing has changed
	fetchOptions := &imap.FetchOptions{
		Flags:        true,
		ChangedSince: currentModSeq,
	}
	messages, err := client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("Fetch() with CHANGEDSINCE = %v", err)
	}

	// No messages should be returned since nothing has changed
	if len(messages) != 0 {
		t.Errorf("Fetch() with CHANGEDSINCE returned %d messages, want 0", len(messages))
	}

	// Now modify the message
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagSeen},
	}
	storeCmd := client.Store(seqSet, &storeFlags, nil)
	storeResults, err := storeCmd.Collect()
	if err != nil {
		t.Fatalf("Store() = %v", err)
	}
	t.Logf("Store results: %d messages", len(storeResults))

	// Fetch the current ModSeq again to verify it changed
	secondFetch, err := client.Fetch(seqSet, &imap.FetchOptions{
		ModSeq: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Second Fetch() = %v", err)
	}
	newModSeq := secondFetch[0].ModSeq
	t.Logf("New ModSeq after flag change: %d", newModSeq)

	// Now fetch again with the old modseq - should return the message
	messages, err = client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("Fetch() with CHANGEDSINCE after change = %v", err)
	}
	t.Logf("Messages returned after change: %d", len(messages))
	if len(messages) != 1 {
		t.Errorf("Fetch() with CHANGEDSINCE after change returned %d messages, want 1", len(messages))
	}
}

func TestStore_UnchangedSince(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	// First, get current ModSeq
	seqSet := imap.SeqSetNum(1)
	firstFetch, err := client.Fetch(seqSet, &imap.FetchOptions{
		ModSeq: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Initial Fetch() = %v", err)
	}
	currentModSeq := firstFetch[0].ModSeq

	// Now modify the message using UNCHANGEDSINCE with the current ModSeq
	// This should succeed because the message hasn't been modified
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagSeen},
	}
	storeOptions := &imap.StoreOptions{
		UnchangedSince: currentModSeq,
	}
	messages, err := client.Store(seqSet, &storeFlags, storeOptions).Collect()
	if err != nil {
		t.Fatalf("Store() with UNCHANGEDSINCE = %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Store() with UNCHANGEDSINCE returned %d messages, want 1", len(messages))
	}

	// Get the new ModSeq
	secondFetch, err := client.Fetch(seqSet, &imap.FetchOptions{
		ModSeq: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Second Fetch() = %v", err)
	}
	newModSeq := secondFetch[0].ModSeq

	// The ModSeq should have increased
	if newModSeq <= currentModSeq {
		t.Errorf("ModSeq after update = %d, want > %d", newModSeq, currentModSeq)
	}

	// Try to modify again with the old ModSeq
	// This should not modify the message because it has changed since
	storeFlags = imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	storeOptions = &imap.StoreOptions{
		UnchangedSince: currentModSeq, // Use the old ModSeq
	}
	messages, err = client.Store(seqSet, &storeFlags, storeOptions).Collect()
	if err != nil {
		t.Fatalf("Second Store() with UNCHANGEDSINCE = %v", err)
	}

	// The operation should not have modified any messages
	if len(messages) != 0 {
		t.Errorf("Second Store() with UNCHANGEDSINCE returned %d messages, should be 0", len(messages))
	}
}
func TestStatus_HighestModSeq(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Test STATUS with HIGHESTMODSEQ parameter
	options := &imap.StatusOptions{
		HighestModSeq: true,
	}
	data, err := client.Status("INBOX", options).Wait()
	if err != nil {
		t.Fatalf("Status() with HIGHESTMODSEQ = %v", err)
	}

	// Verify that HighestModSeq is returned
	if data.HighestModSeq == 0 {
		t.Errorf("StatusData.HighestModSeq is 0, expected non-zero value")
	}
	t.Logf("Mailbox HIGHESTMODSEQ from STATUS: %d", data.HighestModSeq)
}

func TestSearch_ModSeq(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	// First, get current ModSeq for our message
	seqSet := imap.SeqSetNum(1)
	firstFetch, err := client.Fetch(seqSet, &imap.FetchOptions{
		ModSeq: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Initial Fetch() = %v", err)
	}
	currentModSeq := firstFetch[0].ModSeq
	t.Logf("Initial ModSeq: %d", currentModSeq)

	// Now search with MODSEQ criterion using a value lower than current
	// This should find the message
	searchCriteria := &imap.SearchCriteria{
		ModSeq: &imap.SearchCriteriaModSeq{
			ModSeq: currentModSeq - 1,
		},
	}
	searchOptions := &imap.SearchOptions{
		ReturnCount: true,
	}
	results, err := client.Search(searchCriteria, searchOptions).Wait()
	if err != nil {
		t.Fatalf("Search with MODSEQ = %v", err)
	}

	// There should be one message that matches
	if results.Count != 1 {
		t.Errorf("Search with MODSEQ < current returned %d messages, want 1", results.Count)
	}

	// Now search with MODSEQ criterion using current value
	// This should find the message (since MODSEQ criterion is >= not >)
	searchCriteria = &imap.SearchCriteria{
		ModSeq: &imap.SearchCriteriaModSeq{
			ModSeq: currentModSeq,
		},
	}
	results, err = client.Search(searchCriteria, searchOptions).Wait()
	if err != nil {
		t.Fatalf("Search with MODSEQ = %v", err)
	}

	// There should be one message that matches
	if results.Count != 1 {
		t.Errorf("Search with MODSEQ = current returned %d messages, want 1", results.Count)
	}

	// Now search with MODSEQ criterion using a higher value
	// This should NOT find the message
	searchCriteria = &imap.SearchCriteria{
		ModSeq: &imap.SearchCriteriaModSeq{
			ModSeq: currentModSeq + 1,
		},
	}
	results, err = client.Search(searchCriteria, searchOptions).Wait()
	if err != nil {
		t.Fatalf("Search with MODSEQ = %v", err)
	}

	// There should be no messages that match
	if results.Count != 0 {
		t.Errorf("Search with MODSEQ > current returned %d messages, want 0", results.Count)
	}
}

func TestCapability_CondStore(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateNotAuthenticated)
	defer client.Close()
	defer server.Close()

	// Check capabilities after connecting
	capCmd := client.Capability()
	caps, err := capCmd.Wait()
	if err != nil {
		t.Fatalf("Capability() = %v", err)
	}

	_, hasCondStore := caps[imap.CapCondStore]
	if hasCondStore {
		t.Errorf("CapCondStore should not be available before authentication")
	}

	// Login
	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("Login() = %v", err)
	}

	// Check capabilities after login
	capCmd = client.Capability()
	caps, err = capCmd.Wait()
	if err != nil {
		t.Fatalf("Capability() after login = %v", err)
	}

	_, hasCondStore = caps[imap.CapCondStore]
	if !hasCondStore {
		t.Errorf("CapCondStore should be available after authentication")
	} else {
		t.Logf("CONDSTORE capability correctly announced after authentication")
	}
}
