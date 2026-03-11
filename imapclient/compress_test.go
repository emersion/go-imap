package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestCompress(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if algos := client.Caps().CompressAlgorithms(); len(algos) == 0 {
		t.Skipf("COMPRESS not supported")
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop().Wait() = %v", err)
	}
}
