package imapclient_test

import (
	"bufio"
	"fmt"
	"net"
	"slices"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func TestSelect(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	data, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() = %v", err)
	} else if data.NumMessages != 1 {
		t.Errorf("SelectData.NumMessages = %v, want %v", data.NumMessages, 1)
	}
}

// Gmail accepts keywords containing "]" and lists them in FLAGS and
// PERMANENTFLAGS for every mailbox, so rejecting them fails every SELECT.
func TestSelect_keywordWithBracket(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	go func() {
		fmt.Fprint(serverConn, "* OK ready\r\n")
		r := bufio.NewReader(serverConn)
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		tag, _, _ := strings.Cut(line, " ")
		fmt.Fprint(serverConn, "* FLAGS (\\Seen [Imap]/Todo)\r\n")
		fmt.Fprint(serverConn, "* OK [PERMANENTFLAGS (\\Seen [Imap]/Todo \\*)] Flags permitted.\r\n")
		fmt.Fprint(serverConn, "* 1 EXISTS\r\n")
		fmt.Fprintf(serverConn, "%s OK [READ-WRITE] SELECT completed\r\n", tag)
	}()

	client := imapclient.New(clientConn, nil)
	defer client.Close()

	data, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() = %v", err)
	}
	if !slices.Contains(data.Flags, imap.Flag("[Imap]/Todo")) {
		t.Errorf("SelectData.Flags = %v, want it to contain [Imap]/Todo", data.Flags)
	}
	if !slices.Contains(data.PermanentFlags, imap.Flag("[Imap]/Todo")) {
		t.Errorf("SelectData.PermanentFlags = %v, want it to contain [Imap]/Todo", data.PermanentFlags)
	}
}
