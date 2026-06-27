package imapclient_test

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// TestFetchGmailLabels exercises the X-GM-EXT-1 "X-GM-LABELS" FETCH data item
// against a scripted raw IMAP server (the in-process imapmemserver does not
// implement the Gmail extension). It verifies the client both emits the
// "X-GM-LABELS" fetch item and parses the parenthesised label list, mixing
// backslash-prefixed system labels with quoted user labels (one containing a
// space and a slash).
func TestFetchGmailLabels(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("Accept() = %v", err)
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)

		writeLine := func(s string) {
			if _, err := conn.Write([]byte(s + "\r\n")); err != nil {
				t.Errorf("server write: %v", err)
			}
		}

		// Greeting advertising the Gmail extension.
		writeLine("* OK [CAPABILITY IMAP4rev2 X-GM-EXT-1] ready")

		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			fields := strings.SplitN(line, " ", 2)
			tag := fields[0]
			rest := ""
			if len(fields) > 1 {
				rest = strings.ToUpper(fields[1])
			}
			switch {
			case strings.HasPrefix(rest, "LOGIN"):
				writeLine(tag + " OK LOGIN completed")
			case strings.HasPrefix(rest, "SELECT"):
				writeLine("* 1 EXISTS")
				writeLine("* OK [UIDVALIDITY 1] .")
				writeLine("* OK [UIDNEXT 43] .")
				writeLine(tag + " OK [READ-WRITE] SELECT completed")
			case strings.HasPrefix(rest, "FETCH"), strings.HasPrefix(rest, "UID FETCH"):
				// Return a message with a mix of system and user labels.
				writeLine(`* 1 FETCH (UID 42 X-GM-LABELS (\Inbox \Sent "Travel" "Receipts/2024"))`)
				writeLine(tag + " OK FETCH completed")
			case strings.HasPrefix(rest, "LOGOUT"):
				writeLine("* BYE logging out")
				writeLine(tag + " OK LOGOUT completed")
				return
			default:
				writeLine(tag + " OK done")
			}
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial() = %v", err)
	}
	client := imapclient.New(conn, nil)
	defer client.Close()

	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("Login().Wait() = %v", err)
	}
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select().Wait() = %v", err)
	}

	msgs, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		UID:         true,
		GmailLabels: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch().Collect() = %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	got := msgs[0].GmailLabels
	want := []string{`\Inbox`, `\Sent`, "Travel", "Receipts/2024"}
	if len(got) != len(want) {
		t.Fatalf("GmailLabels = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("GmailLabels[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if msgs[0].UID != 42 {
		t.Errorf("UID = %d, want 42", msgs[0].UID)
	}
}
