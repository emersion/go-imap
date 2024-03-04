package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/stretchr/testify/assert"
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
		setRights:             imap.AllRights,
		expectedRights:        imap.AllRights,
	},
	{
		name:                  "custom_folder",
		mailbox:               "MyFolder",
		setRightsModification: imap.RightModificationReplace,
		setRights:             "ailw",
		expectedRights:        "ailw",
	},
	{
		name:                  "custom_child_folder",
		mailbox:               "MyFolder.Child",
		setRightsModification: imap.RightModificationReplace,
		setRights:             "alwrd",
		expectedRights:        "alwrd",
	},
	{
		name:                  "add_rights",
		mailbox:               "MyFolder",
		setRightsModification: imap.RightModificationAdd,
		setRights:             "rwic",
		expectedRights:        "ailwrc",
	},
	{
		name:                  "remove_rights",
		mailbox:               "MyFolder",
		setRightsModification: imap.RightModificationRemove,
		setRights:             "iwc",
		expectedRights:        "alr",
	},
	{
		name:                  "empty_rights",
		mailbox:               "MyFolder.Child",
		setRightsModification: imap.RightModificationReplace,
		setRights:             "a",
		expectedRights:        "a",
	},
}

// TestACL runs tests on SetACL and MyRights commands (for now).
func TestACL(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)

	defer client.Close()
	defer server.Close()

	if err := client.Create("MyFolder", nil).Wait(); err != nil {
		t.Fatalf("create MyFolder error: %v", err)
	}

	if err := client.Create("MyFolder.Child", nil).Wait(); err != nil {
		t.Fatalf("create MyFolder.Child error: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// execute SETACL command
			err := client.SetACL(tc.mailbox, testUsername, tc.setRightsModification, tc.setRights).Wait()
			assert.NoErrorf(t, err, "SetACL().Wait() error")

			// execute GETACL command to reset cache on server
			getACLData, err := client.GetACL(tc.mailbox).Wait()
			if assert.NoErrorf(t, err, "GetACL().Wait() error") {
				assert.Equal(t, tc.mailbox, getACLData.Mailbox)
				assert.Truef(t, tc.expectedRights.Equal(getACLData.Rights[testUsername]),
					"expected: %s, got: %s", tc.expectedRights, getACLData.Rights[testUsername])
			}

			// execute MYRIGHTS command
			myRightsData, err := client.MyRights(tc.mailbox).Wait()
			if assert.NoErrorf(t, err, "MyRights().Wait() error") {
				assert.Equal(t, tc.mailbox, myRightsData.Mailbox)
				assert.Truef(t, tc.expectedRights.Equal(myRightsData.Rights),
					"expected: %s, got: %s", tc.expectedRights, myRightsData.Rights)
			}
		})
	}

	t.Run("set_invalid_rights", func(t *testing.T) {
		assert.Error(t, client.SetACL("MyFolder", testUsername, imap.RightModificationReplace, "bibli").Wait())
		assert.Error(t, client.SetACL("MyFolder", testUsername, imap.RightModificationReplace, "rw i").Wait())
	})

	t.Run("nonexistent_mailbox", func(t *testing.T) {
		assert.Error(t, client.SetACL("BibiMailbox", testUsername, imap.RightModificationReplace, "").Wait())
	})
}
