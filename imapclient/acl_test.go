package imapclient_test

import (
	"testing"

	"github.com/opsxolc/go-imap/v2"
)

// TestACL runs tests on SetACL and MyRights commands (for now).
func TestACL(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)

	defer client.Close()
	defer server.Close()

	client.Create("MyFolder", nil)
	client.Create("MyFolder/Child", nil)

	for i, testcase := range []struct {
		Mailbox   string
		Rights    imap.RightSet
		SetRights imap.RightSet
	}{
		// Testcase 1
		// INBOX
		{
			Mailbox:   "INBOX",
			Rights:    imap.AllRights,
			SetRights: imap.AllRights,
		},
		// Testcase 2
		// Custom folder
		{
			Mailbox:   "MyFolder",
			Rights:    "rwi",
			SetRights: "rwi",
		},
		// Testcase 3
		// Custom child folder
		{
			Mailbox:   "MyFolder/Child",
			Rights:    "rwi",
			SetRights: "rwi",
		},
		// Testcase 4
		// Add rights
		{
			Mailbox:   "MyFolder",
			Rights:    "rwidc",
			SetRights: "+dc",
		},
		// Testcase 5
		// Remove rights
		{
			Mailbox:   "MyFolder",
			Rights:    "rwi",
			SetRights: "-dc",
		},
		// Testcase 6
		// Set empty rights
		{
			Mailbox:   "MyFolder/Child",
			Rights:    "",
			SetRights: "",
		},
	} {
		// execute SETACL command
		err := client.SetACL(testcase.Mailbox, testUsername, testcase.SetRights).Wait()
		if err != nil {
			t.Fatalf("[%v] SetACL().Wait() error: %v", i+1, err)
		}

		// execute MYRIGHTS command
		data, err := client.MyRights(testcase.Mailbox).Wait()
		if err != nil {
			t.Fatalf("[%v] MyRights().Wait() error: %v", i+1, err)
		}

		if data.Mailbox != testcase.Mailbox || data.Rights != testcase.Rights {
			t.Fatalf("[%v] client received incorrect mailbox or rights: %v - %v, %v - %v", i+1,
				testcase.Mailbox, data.Mailbox, testcase.Rights, data.Rights)
		}
	}

	// invalid rights
	if err := client.SetACL("MyFolder", testUsername, "bibli").Wait(); err == nil {
		t.Fatalf("[6] SetACL expected error")
	}
	if err := client.SetACL("MyFolder", testUsername, "rw i").Wait(); err == nil {
		t.Fatalf("[7] SetACL expected error")
	}

	// nonexistent mailbox
	if err := client.SetACL("BibiMailbox", testUsername, "rwi").Wait(); err == nil {
		t.Fatalf("[7] SetACL expected error")
	}
	if _, err := client.MyRights("BibiMailbox").Wait(); err == nil {
		t.Fatalf("[8] SetACL expected error")
	}
}
