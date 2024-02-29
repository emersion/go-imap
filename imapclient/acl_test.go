package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/stretchr/testify/assert"
)

// order matters
var testCases = []struct {
	name                  string
	data                  *imap.MyRightsData
	setRightsModification imap.RightModification
	setRights             imap.RightSet
}{
	{
		name: "inbox",
		data: &imap.MyRightsData{
			Mailbox: "INBOX",
			Rights:  imap.AllRights,
		},
		setRightsModification: imap.RightModificationReplace,
		setRights:             imap.AllRights,
	},
	{
		name: "custom_folder",
		data: &imap.MyRightsData{
			Mailbox: "MyFolder",
			Rights:  "rwi",
		},
		setRightsModification: imap.RightModificationReplace,
		setRights:             "rwi",
	},
	{
		name: "custom_child_folder",
		data: &imap.MyRightsData{
			Mailbox: "MyFolder/Child",
			Rights:  "rwi",
		},
		setRightsModification: imap.RightModificationReplace,
		setRights:             "rwi",
	},
	{
		name: "add_rights",
		data: &imap.MyRightsData{
			Mailbox: "MyFolder",
			Rights:  "rwidc",
		},
		setRightsModification: imap.RightModificationAdd,
		setRights:             "dc",
	},
	{
		name: "remove_rights",
		data: &imap.MyRightsData{
			Mailbox: "MyFolder",
			Rights:  "rwi",
		},
		setRightsModification: imap.RightModificationRemove,
		setRights:             "dc",
	},
	{
		name: "empty_rights",
		data: &imap.MyRightsData{
			Mailbox: "MyFolder/Child",
			Rights:  "",
		},
		setRightsModification: imap.RightModificationReplace,
		setRights:             "",
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
	if err := client.Create("MyFolder/Child", nil).Wait(); err != nil {
		t.Fatalf("create MyFolder/Child error: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// execute SETACL command
			err := client.SetACL(tc.data.Mailbox, testUsername, tc.setRightsModification, tc.setRights).Wait()
			assert.NoErrorf(t, err, "SetACL().Wait() error")

			// execute MYRIGHTS command
			data, err := client.MyRights(tc.data.Mailbox).Wait()
			assert.NoErrorf(t, err, "MyRights().Wait() error")
			assert.Equal(t, tc.data, data)
		})
	}

	t.Run("set_invalid_rights", func(t *testing.T) {
		assert.Error(t, client.SetACL("MyFolder", testUsername, imap.RightModificationReplace, "bibli").Wait())
		assert.Error(t, client.SetACL("MyFolder", testUsername, imap.RightModificationReplace, "rw i").Wait())
	})

	t.Run("nonexistent_mailbox", func(t *testing.T) {
		assert.Error(t, client.SetACL("BibiMailbox", testUsername, imap.RightModificationReplace, "").Wait())
		_, err := client.MyRights("BibiMailbox").Wait()
		assert.Error(t, err)
	})
}
