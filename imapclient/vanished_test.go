package imapclient_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// TestVanished checks that a VANISHED (EARLIER) response (RFC 7162) is parsed and
// its expunged UIDs delivered to UnilateralDataHandler.Vanished. A minimal raw
// server is used because imapmemserver does not implement QRESYNC.
func TestVanished(t *testing.T) {
	clientConn, serverConn := net.Pipe()

	var (
		mu         sync.Mutex
		gotUIDs    string
		gotEarlier bool
		called     bool
	)
	client := imapclient.New(clientConn, &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Vanished: func(uids imap.UIDSet, earlier bool) {
				mu.Lock()
				gotUIDs, gotEarlier, called = uids.String(), earlier, true
				mu.Unlock()
			},
		},
	})
	defer client.Close()

	// Minimal server: greeting, then for the client's first command emit a
	// VANISHED (EARLIER) response followed by the tagged completion.
	go func() {
		br := bufio.NewReader(serverConn)
		_, _ = io.WriteString(serverConn, "* OK [CAPABILITY IMAP4rev2] ready\r\n")
		line, _ := br.ReadString('\n')
		tag, _, _ := strings.Cut(line, " ")
		_, _ = io.WriteString(serverConn, "* VANISHED (EARLIER) 1:3,7\r\n")
		_, _ = io.WriteString(serverConn, tag+" OK completed\r\n")
	}()

	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop() = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !called {
		t.Fatal("Vanished handler was not called")
	}
	if !gotEarlier {
		t.Error("earlier = false, want true (VANISHED (EARLIER))")
	}
	if gotUIDs != "1:3,7" {
		t.Errorf("vanished UIDs = %q, want %q", gotUIDs, "1:3,7")
	}
}

// TestFetchVanishedModifier checks that FetchOptions.Vanished adds the VANISHED
// modifier to a UID FETCH with CHANGEDSINCE.
func TestFetchVanishedModifier(t *testing.T) {
	clientConn, serverConn := net.Pipe()

	client := imapclient.New(clientConn, nil)
	defer client.Close()

	gotCmd := make(chan string, 1)
	go func() {
		br := bufio.NewReader(serverConn)
		_, _ = io.WriteString(serverConn, "* OK ready\r\n")
		line, _ := br.ReadString('\n')
		gotCmd <- line
		tag, _, _ := strings.Cut(line, " ")
		_, _ = io.WriteString(serverConn, tag+" OK completed\r\n")
	}()

	err := client.Fetch(imap.UIDSetNum(1, 2), &imap.FetchOptions{
		UID:          true,
		Flags:        true,
		ChangedSince: 100,
		Vanished:     true,
	}).Close()
	if err != nil {
		t.Fatalf("Fetch() = %v", err)
	}

	cmd := <-gotCmd
	if !strings.Contains(cmd, "(CHANGEDSINCE 100 VANISHED)") {
		t.Errorf("FETCH command = %q, want it to contain %q", strings.TrimSpace(cmd), "(CHANGEDSINCE 100 VANISHED)")
	}
}
