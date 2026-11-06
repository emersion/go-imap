package imapclient_test

import (
	"bufio"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

const literalTestTimeout = 5 * time.Second

func TestFetchCollectInterruptedLiteral(t *testing.T) {
	tests := []struct {
		name     string
		response string
		options  imap.FetchOptions
	}{
		{
			name:     "body",
			response: "* 1 FETCH (BODY[] {8}\r\nshort",
			options:  imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{{}}},
		},
		{
			name:     "binary",
			response: "* 1 FETCH (BINARY[] ~{8}\r\nshort",
			options:  imap.FetchOptions{BinarySection: []*imap.FetchItemBinarySection{{}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, local, _ := newLiteralTestClient(t, tt.response, false, true)
			cmd := client.Fetch(imap.SeqSetNum(1), &tt.options)
			t.Cleanup(func() {
				_ = local.Close()
				_ = cmd.Close()
			})
			messages, err := cmd.Collect()
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("Collect() error = %v; want %v", err, io.ErrUnexpectedEOF)
			}
			if len(messages) != 0 {
				t.Fatalf("Collect() returned %v incomplete messages", len(messages))
			}
		})
	}
}

func TestFetchLiteralReadInterrupted(t *testing.T) {
	tests := []struct {
		name      string
		interrupt func(local, peer net.Conn) error
		wantErr   error
	}{
		{name: "peer_eof", interrupt: func(_, peer net.Conn) error { return peer.Close() }, wantErr: io.ErrUnexpectedEOF},
		{name: "local_close", interrupt: func(local, _ net.Conn) error { return local.Close() }, wantErr: io.ErrClosedPipe},
		{name: "read_deadline", interrupt: func(local, _ net.Conn) error { return local.SetReadDeadline(time.Now()) }, wantErr: os.ErrDeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, local, peer := newLiteralTestClient(t, "* 1 FETCH (BODY[] {8}\r\nshort", false, false)
			cmd := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{{}}})
			t.Cleanup(func() {
				_ = local.Close()
				_ = cmd.Close()
			})
			msg := cmd.Next()
			if msg == nil {
				t.Fatal("Next() returned no message")
			}
			item, ok := msg.Next().(imapclient.FetchItemDataBodySection)
			if !ok || item.Literal == nil {
				t.Fatal("Next() returned no BODY literal")
			}
			prefix := make([]byte, len("short"))
			if _, err := io.ReadFull(item.Literal, prefix); err != nil || string(prefix) != "short" {
				t.Fatalf("ReadFull() = %q, %v; want short, nil", prefix, err)
			}
			if err := tt.interrupt(local, peer); err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadAll(item.Literal); !errors.Is(err, tt.wantErr) {
				t.Fatalf("ReadAll() error = %v; want %v", err, tt.wantErr)
			}
			select {
			case <-client.Closed():
			case <-time.After(time.Second):
				t.Fatal("decoder did not stop after a literal read error")
			}
			if err := cmd.Close(); err == nil {
				t.Fatal("Close() succeeded after an interrupted literal")
			}
		})
	}
}

func TestFetchLiteralComplete(t *testing.T) {
	tests := []struct {
		name     string
		response string
		body     string
	}{
		{name: "literal", response: "* 1 FETCH (BODY[] {5}\r\nhello)\r\n", body: "hello"},
		{name: "empty_literal", response: "* 1 FETCH (BODY[] {0}\r\n)\r\n"},
		{name: "quoted_string", response: "* 1 FETCH (BODY[] \"hello\")\r\n", body: "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, local, _ := newLiteralTestClient(t, tt.response, true, false)
			section := &imap.FetchItemBodySection{}
			cmd := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{section}})
			t.Cleanup(func() {
				_ = local.Close()
				_ = cmd.Close()
			})
			messages, err := cmd.Collect()
			if err != nil || len(messages) != 1 {
				t.Fatalf("Collect() returned %v messages, %v; want 1, nil", len(messages), err)
			}
			if body := string(messages[0].FindBodySection(section)); body != tt.body {
				t.Errorf("BODY = %q; want %q", body, tt.body)
			}
		})
	}
}

func newLiteralTestClient(t *testing.T, response string, tagged, closePeer bool) (*imapclient.Client, net.Conn, net.Conn) {
	t.Helper()
	local, peer := net.Pipe()
	if err := peer.SetDeadline(time.Now().Add(literalTestTimeout)); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var serverErr error
	go func() {
		defer close(done)
		defer peer.Close()
		if _, err := io.WriteString(peer, "* PREAUTH [CAPABILITY IMAP4rev2 BINARY] ready\r\n"); err != nil {
			serverErr = err
			return
		}
		command, err := bufio.NewReader(peer).ReadString('\n')
		if err != nil {
			serverErr = err
			return
		}
		if tagged {
			response += strings.SplitN(command, " ", 2)[0] + " OK done\r\n"
		}
		if _, err := io.WriteString(peer, response); err != nil {
			serverErr = err
			return
		}
		if !closePeer {
			_, _ = io.Copy(io.Discard, peer)
		}
	}()
	client := imapclient.New(local, nil)
	t.Cleanup(func() {
		_ = local.Close()
		_ = peer.Close()
		_ = client.Close()
		select {
		case <-done:
			if serverErr != nil {
				t.Errorf("test server failed: %v", serverErr)
			}
		case <-time.After(literalTestTimeout):
			t.Error("test server did not stop")
		}
	})
	return client, local, peer
}
