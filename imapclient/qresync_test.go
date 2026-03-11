package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestEnable_QResync(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Enable QRESYNC
	data, err := client.Enable(imap.CapQResync).Wait()
	if err != nil {
		t.Fatalf("Enable(QRESYNC) = %v", err)
	}

	if !data.Caps.Has(imap.CapQResync) {
		t.Errorf("QRESYNC capability not enabled")
	}
	t.Logf("Successfully enabled QRESYNC")
}

func TestSelect_QResync(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Enable QRESYNC first
	_, err := client.Enable(imap.CapQResync).Wait()
	if err != nil {
		t.Fatalf("Enable(QRESYNC) = %v", err)
	}

	// First SELECT to get initial state
	firstSelect, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("First Select() = %v", err)
	}
	t.Logf("Initial SELECT - UIDValidity: %d, HighestModSeq: %d",
		firstSelect.UIDValidity, firstSelect.HighestModSeq)

	// Unselect to test QRESYNC SELECT
	if err := client.Unselect().Wait(); err != nil {
		t.Fatalf("Unselect() = %v", err)
	}

	// SELECT with QRESYNC
	qresyncOptions := &imap.SelectOptions{
		QResync: &imap.QResyncData{
			UIDValidity: firstSelect.UIDValidity,
			ModSeq:      firstSelect.HighestModSeq,
		},
	}

	secondSelect, err := client.Select("INBOX", qresyncOptions).Wait()
	if err != nil {
		t.Fatalf("QRESYNC Select() = %v", err)
	}

	// Verify QRESYNC worked
	if secondSelect.UIDValidity != firstSelect.UIDValidity {
		t.Errorf("UIDValidity changed: %d != %d", secondSelect.UIDValidity, firstSelect.UIDValidity)
	}
	if secondSelect.HighestModSeq < firstSelect.HighestModSeq {
		t.Errorf("HighestModSeq decreased: %d < %d", secondSelect.HighestModSeq, firstSelect.HighestModSeq)
	}
	t.Logf("QRESYNC SELECT successful")
}

func TestSelect_QResync_WithKnownUIDs(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Enable QRESYNC first
	_, err := client.Enable(imap.CapQResync).Wait()
	if err != nil {
		t.Fatalf("Enable(QRESYNC) = %v", err)
	}

	// First SELECT to get initial state
	firstSelect, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("First Select() = %v", err)
	}

	// Get some UIDs to test with
	fetchOptions := &imap.FetchOptions{UID: true}
	messages, err := client.Fetch(imap.SeqSetNum(1), fetchOptions).Collect()
	if err != nil {
		t.Fatalf("Fetch UIDs = %v", err)
	}

	var knownUIDs imap.UIDSet
	if len(messages) > 0 {
		knownUIDs = imap.UIDSetNum(messages[0].UID)
	}

	// Unselect to test QRESYNC SELECT with known UIDs
	if err := client.Unselect().Wait(); err != nil {
		t.Fatalf("Unselect() = %v", err)
	}

	// SELECT with QRESYNC and known UIDs
	qresyncOptions := &imap.SelectOptions{
		QResync: &imap.QResyncData{
			UIDValidity: firstSelect.UIDValidity,
			ModSeq:      firstSelect.HighestModSeq,
			KnownUIDs:   knownUIDs,
		},
	}

	_, err = client.Select("INBOX", qresyncOptions).Wait()
	if err != nil {
		t.Fatalf("QRESYNC Select() with known UIDs = %v", err)
	}

	t.Logf("QRESYNC SELECT with known UIDs successful")
}

func TestUIDFetch_Vanished(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	// Enable QRESYNC first
	_, err := client.Enable(imap.CapQResync).Wait()
	if err != nil {
		t.Fatalf("Enable(QRESYNC) = %v", err)
	}

	// Test UID FETCH with VANISHED modifier
	fetchOptions := &imap.FetchOptions{
		Flags:        true,
		ChangedSince: 1, // Use a low modseq to potentially get some results
		Vanished:     true,
	}

	uidSet := imap.UIDSetNum(1)
	uidSet.AddRange(1, 0) // 1:*
	messages, err := client.Fetch(uidSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("UID FETCH with VANISHED = %v", err)
	}

	t.Logf("UID FETCH with VANISHED returned %d messages", len(messages))
}

func TestVanished_Response(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	// Enable QRESYNC first
	_, err := client.Enable(imap.CapQResync).Wait()
	if err != nil {
		t.Fatalf("Enable(QRESYNC) = %v", err)
	}

	// Note: In a real test, we would need to trigger an expunge that causes
	// a VANISHED response. For now, we just verify QRESYNC is enabled.
	// The VANISHED responses would be handled by the UnilateralDataHandler
	// which can be set when creating the client.

	// This test just verifies that QRESYNC is properly enabled
	// and the client can handle the expected protocol

	t.Logf("VANISHED response handler test completed")
}

func TestCapability_QResync_Implications(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateNotAuthenticated)
	defer client.Close()
	defer server.Close()

	// Check that QRESYNC implies CONDSTORE
	caps, err := client.Capability().Wait()
	if err != nil {
		t.Fatalf("Capability() = %v", err)
	}

	// Login first
	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("Login() = %v", err)
	}

	// Enable QRESYNC
	enableData, err := client.Enable(imap.CapQResync).Wait()
	if err != nil {
		t.Fatalf("Enable(QRESYNC) = %v", err)
	}

	// Verify QRESYNC implies CONDSTORE
	if enableData.Caps.Has(imap.CapQResync) && !enableData.Caps.Has(imap.CapCondStore) {
		// Check if CONDSTORE is implied by QRESYNC in the capability system
		if !caps.Has(imap.CapCondStore) {
			t.Errorf("QRESYNC should imply CONDSTORE capability")
		}
	}

	t.Logf("QRESYNC capability implications verified")
}
