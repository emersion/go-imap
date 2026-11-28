package imapclient_test

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestStore(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	seqSet := imap.SeqSetNum(1)
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	msgs, err := client.Store(seqSet, &storeFlags, nil).Collect()
	if err != nil {
		t.Fatalf("Store().Collect() = %v", err)
	} else if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %v, want %v", len(msgs), 1)
	}
	msg := msgs[0]
	if msg.SeqNum != 1 {
		t.Errorf("msg.SeqNum = %v, want %v", msg.SeqNum, 1)
	}

	found := false
	for _, f := range msg.Flags {
		if f == imap.FlagDeleted {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("msg.Flags is missing deleted flag: %v", msg.Flags)
	}
}

func TestStoreUnchangedSince(t *testing.T) {
	// Authenticated so we can observe HIGHESTMODSEQ in SELECT directly.
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapCondStore) {
		t.Skip("CONDSTORE not supported")
	}

	seqSet := imap.SeqSetNum(1)

	// Select mailbox before first STORE, get HIGHESTMODSEQ=1.
	selectData, err := client.Select("INBOX", &imap.SelectOptions{
		CondStore: true,
	}).Wait()
	if err != nil {
		t.Fatalf("client.Select(): %v", err)
	}
	originalModSeq := selectData.HighestModSeq

	// STORE sets MODSEQ for message to 2.
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	cmd := client.Store(seqSet, &storeFlags, &imap.StoreOptions{
		UnchangedSince: originalModSeq,
	})
	msgs, err := cmd.Collect()
	if err != nil {
		t.Fatalf("Store().Collect() = %v", err)
	} else if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %v, want %v", len(msgs), 1)
	}
	msg := msgs[0]
	if msg.SeqNum != 1 {
		t.Errorf("msg.SeqNum = %v, want %v", msg.SeqNum, 1)
	}
	newModSeq := msg.ModSeq
	if newModSeq == originalModSeq {
		t.Errorf("modseq is not advanced, msg.ModSeq = %v, want %v", msg.ModSeq, originalModSeq)
	}
	data := cmd.Data()
	if data != nil && data.Modified != nil {
		t.Errorf("data.Modified = %v, want nil", data.Modified)
	}

	// Second STORE with UNCHANGEDSINCE=1 will skip message.
	storeFlags = imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	cmd = client.Store(seqSet, &storeFlags, &imap.StoreOptions{
		UnchangedSince: originalModSeq,
	})
	msgs, err = cmd.Collect()
	if err != nil {
		t.Fatalf("Store().Collect() = %v", err)
	} else if len(msgs) != 0 {
		t.Fatalf("len(msgs) = %v, want %v", len(msgs), 0)
	}
	data = cmd.Data()
	if data == nil {
		t.Fatalf("cmd.Data() = %v, want non-nil", data)
	}
	modifiedSeq, ok := data.Modified.(imap.SeqSet)
	if !ok {
		t.Fatalf("data.Modified = %v, want imap.SeqSet", data)
	}
	modifiedNums, _ := modifiedSeq.Nums()
	if !reflect.DeepEqual(modifiedNums, []uint32{1}) {
		t.Errorf("modifiedSeq.Nums() = %v, want %v", modifiedNums, []uint32{1})
	}
}
