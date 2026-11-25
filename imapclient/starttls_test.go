package imapclient_test

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

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

func TestStartTLSMalformedLineEnd(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		r := bufio.NewReader(conn)
		conn.Write([]byte("* OK ready\r\n"))
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(line, "T1 STARTTLS") {
				// Complete the tagged command with a malformed line end:
				// the command succeeds, but the decoder cannot consume CRLF.
				conn.Write([]byte("T1 OK go\rZ\n"))
				return
			}
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial() = %v", err)
	}
	defer conn.Close()

	result := make(chan error, 1)
	go func() {
		_, err := imapclient.NewStartTLS(conn, &imapclient.Options{
			TLSConfig: &tls.Config{InsecureSkipVerify: true},
		})
		result <- err
	}()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("NewStartTLS() unexpectedly succeeded")
		}
		if !strings.Contains(fmt.Sprint(err), "CRLF") {
			t.Fatalf("NewStartTLS() error = %v, want CRLF error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("NewStartTLS() blocked forever after malformed tagged OK")
	}

	<-done
}
