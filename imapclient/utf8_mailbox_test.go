package imapclient_test

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// scriptedServer answers client commands with canned responses and records
// the commands it receives.
func scriptedServer(t *testing.T, caps string, respond func(tag, line string) []string) (*imapclient.Client, <-chan string) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close(); serverConn.Close() })
	received := make(chan string, 64)
	go func() {
		defer close(received)
		r := bufio.NewReader(serverConn)
		write := func(s string) bool {
			_, err := serverConn.Write([]byte(s + "\r\n"))
			return err == nil
		}
		if !write("* OK [CAPABILITY " + caps + "] ready") {
			return
		}
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			received <- line
			tag, rest, _ := strings.Cut(line, " ")
			for _, resp := range respond(tag, rest) {
				if !write(resp) {
					return
				}
			}
			if !write(tag + " OK done") {
				return
			}
		}
	}()
	return imapclient.New(clientConn, nil), received
}

func dualRevResponder(tag, cmd string) []string {
	switch {
	case strings.HasPrefix(cmd, "ENABLE"):
		return []string{"* ENABLED IMAP4rev2"}
	case strings.HasPrefix(cmd, "LIST"):
		return []string{
			`* LIST () "/" "you & i"`,
			`* LIST () "/" "R&D/Größe"`,
		}
	}
	return nil
}

func TestUTF8MailboxNamesAfterEnableIMAP4rev2(t *testing.T) {
	client, received := scriptedServer(t, "IMAP4rev1 IMAP4rev2 AUTH=PLAIN", dualRevResponder)
	defer client.Close()

	if _, err := client.Enable(imap.CapIMAP4rev2).Wait(); err != nil {
		t.Fatalf("Enable() = %v", err)
	}
	<-received // ENABLE

	mailboxes, err := client.List("", "*", nil).Collect()
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	<-received // LIST
	var names []string
	for _, mbox := range mailboxes {
		names = append(names, mbox.Mailbox)
	}
	if want := []string{"you & i", "R&D/Größe"}; strings.Join(names, "|") != strings.Join(want, "|") {
		t.Errorf("List() names = %q, want %q", names, want)
	}

	if _, err := client.Status("R&D/Größe", &imap.StatusOptions{NumMessages: true}).Wait(); err != nil {
		t.Fatalf("Status() = %v", err)
	}
	if got, want := <-received, `STATUS "R&D/Größe" (MESSAGES)`; !strings.HasSuffix(got, want) {
		t.Errorf("STATUS command = %q, want suffix %q", got, want)
	}
}

func TestModifiedUTF7MailboxNamesWithoutEnable(t *testing.T) {
	client, received := scriptedServer(t, "IMAP4rev1 AUTH=PLAIN", func(tag, cmd string) []string {
		if strings.HasPrefix(cmd, "LIST") {
			return []string{`* LIST () "/" "you &- i"`, `* LIST () "/" "Gr&APYA3w-e"`}
		}
		return nil
	})
	defer client.Close()

	mailboxes, err := client.List("", "*", nil).Collect()
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	<-received
	if len(mailboxes) != 2 || mailboxes[0].Mailbox != "you & i" || mailboxes[1].Mailbox != "Größe" {
		t.Errorf("List() = %+v, want [you & i, Größe]", mailboxes)
	}

	if _, err := client.Status("R&D", &imap.StatusOptions{NumMessages: true}).Wait(); err != nil {
		t.Fatalf("Status() = %v", err)
	}
	if got, want := <-received, `STATUS "R&-D" (MESSAGES)`; !strings.HasSuffix(got, want) {
		t.Errorf("STATUS command = %q, want suffix %q", got, want)
	}
}
