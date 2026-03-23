package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestExpunge(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	seqNums, err := client.Expunge().Collect()
	if err != nil {
		t.Fatalf("Expunge() = %v", err)
	} else if len(seqNums) != 0 {
		t.Errorf("Expunge().Collect() = %v, want []", seqNums)
	}

	seqSet := imap.SeqSetNum(1)
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	if err := client.Store(seqSet, &storeFlags, nil).Close(); err != nil {
		t.Fatalf("Store() = %v", err)
	}

	seqNums, err = client.Expunge().Collect()
	if err != nil {
		t.Fatalf("Expunge() = %v", err)
	} else if len(seqNums) != 1 || seqNums[0] != 1 {
		t.Errorf("Expunge().Collect() = %v, want [1]", seqNums)
	}
}
