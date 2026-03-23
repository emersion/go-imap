package imapclient_test

import (
	"crypto/tls"
	"testing"

	"github.com/emersion/go-imap/v2/imapclient"
)

func TestStartTLS(t *testing.T) {
	conn, server := newMemClientServerPair(t)
	defer conn.Close()
	defer server.Close()

	options := imapclient.Options{
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client, err := imapclient.NewStartTLS(conn, &options)
	if err != nil {
		t.Fatalf("NewStartTLS() = %v", err)
	}
	defer client.Close()

	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop().Wait() = %v", err)
	}
}
