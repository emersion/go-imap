package imapclient_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
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
	storeCmd := client.Store(seqSet, &storeFlags, nil)
	msgs, err := storeCmd.Collect()
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
	if modified := storeCmd.ModifiedUIDs(); modified != nil {
		t.Errorf("ModifiedUIDs() = %v without MODIFIED response, want nil", modified)
	}
}

func TestStoreModifiedUIDs(t *testing.T) {
	for _, test := range []struct {
		name     string
		numSet   imap.NumSet
		response string
		set      string
	}{
		{name: "UID singleton", numSet: imap.UIDSetNum(7), response: "7", set: "7"},
		{name: "UID range", numSet: imap.UIDSetNum(7, 8, 9, 11), response: "7:9,11", set: "7:9,11"},
		{name: "sequence number", numSet: imap.SeqSetNum(7), response: "7"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd, command := rawStoreCommand(t, test.numSet, "OK [MODIFIED "+test.response+"] completed")
			if err := cmd.Close(); err != nil {
				t.Fatalf("Store().Close() = %v", err)
			}
			if got := strings.TrimSpace(<-command); !strings.Contains(got, "STORE") {
				t.Fatalf("command = %q, want STORE", got)
			}

			modified := cmd.ModifiedUIDs()
			if test.set == "" {
				if modified != nil {
					t.Fatalf("ModifiedUIDs() = %v, want nil for sequence STORE", modified)
				}
				return
			}
			if modified == nil {
				t.Fatal("ModifiedUIDs() = nil, want modified set")
			}
			if got := modified.String(); got != test.set {
				t.Errorf("ModifiedUIDs() = %q, want %q", got, test.set)
			}
			modified[0].Start = 99
			if got := cmd.ModifiedUIDs().String(); got != test.set {
				t.Errorf("ModifiedUIDs() after caller mutation = %q, want %q", got, test.set)
			}
		})
	}
}

func TestStoreModifiedUIDsRejectsTaggedNOAndBAD(t *testing.T) {
	for _, typ := range []string{"NO", "BAD"} {
		t.Run(typ, func(t *testing.T) {
			cmd, command := rawStoreCommand(t, imap.UIDSetNum(7), fmt.Sprintf("%s rejected", typ))
			if err := cmd.Close(); err == nil {
				t.Fatalf("Store().Close() = nil, want tagged %s error", typ)
			} else if imapErr, ok := err.(*imap.Error); !ok || string(imapErr.Type) != typ {
				t.Fatalf("Store().Close() = %T %v, want *imap.Error %s", err, err, typ)
			}
			<-command
			if modified := cmd.ModifiedUIDs(); modified != nil {
				t.Errorf("ModifiedUIDs() = %v after tagged %s, want nil", modified, typ)
			}
		})
	}
}

func TestMoveFallbackPreservesStoreClose(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	client := imapclient.New(clientConn, nil)
	defer client.Close()
	defer serverConn.Close()

	commands := make(chan string, 3)
	go func() {
		reader := bufio.NewReader(serverConn)
		_, _ = io.WriteString(serverConn, "* OK [CAPABILITY IMAP4rev1 UIDPLUS] ready\r\n")
		for i := 0; i < cap(commands); i++ {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			tag, command, ok := strings.Cut(line, " ")
			if !ok {
				return
			}
			commands <- command
			switch {
			case strings.HasPrefix(command, "UID COPY"):
				_, _ = fmt.Fprintf(serverConn, "%s OK [COPYUID 42 7 9] copied\r\n", tag)
			case strings.HasPrefix(command, "UID STORE"):
				_, _ = fmt.Fprintf(serverConn, "%s OK STORE completed\r\n", tag)
			case strings.HasPrefix(command, "UID EXPUNGE"):
				_, _ = fmt.Fprintf(serverConn, "%s OK UID EXPUNGE completed\r\n", tag)
			default:
				_, _ = fmt.Fprintf(serverConn, "%s BAD unexpected command\r\n", tag)
			}
		}
	}()

	data, err := client.Move(imap.UIDSetNum(7), "Archive").Wait()
	if err != nil {
		t.Fatalf("Move().Wait() = %v", err)
	}
	if data.UIDValidity != 42 || data.SourceUIDs.String() != "7" || data.DestUIDs.String() != "9" {
		t.Errorf("MoveData = %#v, want UIDVALIDITY 42, source 7, destination 9", data)
	}
	if got := <-commands; !strings.HasPrefix(got, "UID COPY 7") {
		t.Errorf("first fallback command = %q, want UID COPY", got)
	}
	if got := <-commands; !strings.HasPrefix(got, "UID STORE 7") {
		t.Errorf("second fallback command = %q, want UID STORE", got)
	}
	if got := <-commands; !strings.HasPrefix(got, "UID EXPUNGE 7") {
		t.Errorf("third fallback command = %q, want UID EXPUNGE", got)
	}
}

func rawStoreCommand(t *testing.T, numSet imap.NumSet, response string) (*imapclient.FetchCommand, <-chan string) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	client := imapclient.New(clientConn, nil)
	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	command := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(serverConn)
		_, _ = io.WriteString(serverConn, "* OK ready\r\n")
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		tag, _, ok := strings.Cut(line, " ")
		if !ok {
			return
		}
		command <- line
		_, _ = fmt.Fprintf(serverConn, "%s %s\r\n", tag, response)
	}()

	return client.Store(numSet, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}, nil), command
}
