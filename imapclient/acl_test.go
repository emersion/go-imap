package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

// order matters
var testCases = []struct {
	name                  string
	mailbox               string
	setRightsModification imap.RightModification
	setRights             imap.RightSet
	expectedRights        imap.RightSet
	execStatusCmd         bool
}{
	{
		name:                  "inbox",
		mailbox:               "INBOX",
		setRightsModification: imap.RightModificationReplace,
		setRights:             imap.RightSet("akxeilprwtscd"),
		expectedRights:        imap.RightSet("akxeilprwtscd"),
	},
	{
		name:                  "custom_folder",
		mailbox:               "MyFolder",
		setRightsModification: imap.RightModificationReplace,
		setRights:             imap.RightSet("ailw"),
		expectedRights:        imap.RightSet("ailw"),
	},
	{
		name:                  "custom_child_folder",
		mailbox:               "MyFolder.Child",
		setRightsModification: imap.RightModificationReplace,
		setRights:             imap.RightSet("aelrwtd"),
		expectedRights:        imap.RightSet("aelrwtd"),
	},
	{
		name:                  "add_rights",
		mailbox:               "MyFolder",
		setRightsModification: imap.RightModificationAdd,
		setRights:             imap.RightSet("rwi"),
		expectedRights:        imap.RightSet("ailwr"),
	},
	{
		name:                  "remove_rights",
		mailbox:               "MyFolder",
		setRightsModification: imap.RightModificationRemove,
		setRights:             imap.RightSet("iwc"),
		expectedRights:        imap.RightSet("alr"),
	},
	{
		name:                  "empty_rights",
		mailbox:               "MyFolder.Child",
		setRightsModification: imap.RightModificationReplace,
		setRights:             imap.RightSet("a"),
		expectedRights:        imap.RightSet("a"),
	},
}

// TestACL runs tests on SetACL, GetACL and MyRights commands.
func TestACL(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(imap.CapACL) {
		t.Skipf("server doesn't support ACL")
	}

	if err := client.Create("MyFolder", nil).Wait(); err != nil {
		t.Fatalf("create MyFolder error: %v", err)
	}

	if err := client.Create("MyFolder/Child", nil).Wait(); err != nil {
		t.Fatalf("create MyFolder/Child error: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// execute SETACL command
			err := client.SetACL(tc.mailbox, testUsername, tc.setRightsModification, tc.setRights).Wait()
			if err != nil {
				t.Errorf("SetACL().Wait() error: %v", err)
			}

			// execute GETACL command to reset cache on server
			getACLData, err := client.GetACL(tc.mailbox).Wait()
			if err != nil {
				t.Errorf("GetACL().Wait() error: %v", err)
			}

			if !tc.expectedRights.Equal(getACLData.Rights[testUsername]) {
				t.Errorf("GETACL returned wrong rights; expected: %s, got: %s", tc.expectedRights, getACLData.Rights[testUsername])
			}

			// execute MYRIGHTS command
			myRightsData, err := client.MyRights(tc.mailbox).Wait()
			if err != nil {
				t.Errorf("MyRights().Wait() error: %v", err)
			}

			if !tc.expectedRights.Equal(myRightsData.Rights) {
				t.Errorf("MYRIGHTS returned wrong rights; expected: %s, got: %s", tc.expectedRights, myRightsData.Rights)
			}
		})
	}

	t.Run("nonexistent_mailbox", func(t *testing.T) {
		if client.SetACL("BibiMailbox", testUsername, imap.RightModificationReplace, nil).Wait() == nil {
			t.Errorf("expected error")
		}
	})
}
