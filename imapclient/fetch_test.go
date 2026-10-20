package imapclient_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func TestFetch(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	seqSet := imap.SeqSetNum(1)
	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}
	messages, err := client.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		t.Fatalf("failed to fetch first message: %v", err)
	} else if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}

	msg := messages[0]
	if len(msg.BodySection) != 1 {
		t.Fatalf("len(msg.BodySection) = %v, want 1", len(msg.BodySection))
	}
	b := msg.FindBodySection(bodySection)
	if b == nil {
		t.Fatalf("FindBodySection() = nil")
	}
	body := strings.ReplaceAll(string(b), "\r\n", "\n")
	if body != simpleRawMessage {
		t.Errorf("body mismatch: got \n%v\n but want \n%v", body, simpleRawMessage)
	}
}

// https://github.com/emersion/go-imap/issues/774
func TestFetch_emptyLiteralMissingSP(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	client := imapclient.New(clientConn, nil)
	defer client.Close()

	go func() {
		io.WriteString(serverConn, "* OK [CAPABILITY IMAP4rev1] ready\r\n")
		br := bufio.NewReader(serverConn)
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		tag, _, _ := strings.Cut(line, " ")
		io.WriteString(serverConn, "* 1 FETCH (BODY[HEADER.FIELDS (REFERENCES)] {0}\r\nUID 42)\r\n")
		io.WriteString(serverConn, tag+" OK FETCH completed\r\n")
	}()

	bodySection := &imap.FetchItemBodySection{
		Specifier:    imap.PartSpecifierHeader,
		HeaderFields: []string{"References"},
		Peek:         true,
	}
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch().Collect() = %v", err)
	} else if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
	if msg := messages[0]; msg.UID != 42 {
		t.Errorf("UID = %v, want 42", msg.UID)
	}
}
