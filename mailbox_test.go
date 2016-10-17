package imap_test

import (
	"fmt"
	"sort"
	"reflect"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/internal"
)

func TestCanonicalMailboxName(t *testing.T) {
	if got := imap.CanonicalMailboxName("Inbox"); got != imap.InboxName {
		t.Errorf("Invalid canonical mailbox name: expected %q but got %q", imap.InboxName, got)
	}
	if got := imap.CanonicalMailboxName("Drafts"); got != "Drafts" {
		t.Errorf("Invalid canonical mailbox name: expected %q but got %q", "Drafts", got)
	}
}

func TestMailboxInfo(t *testing.T) {
	fields := []interface{}{
		[]interface{}{"\\Noselect", "\\Recent", "\\Unseen"},
		"/",
		"INBOX",
	}
	info := &imap.MailboxInfo{
		Attributes: []string{"\\Noselect", "\\Recent", "\\Unseen"},
		Delimiter:  "/",
		Name:       "INBOX",
	}

	testMailboxInfo_Parse(t, fields, info)
	testMailboxInfo_Format(t, info, fields)
}

func testMailboxInfo_Parse(t *testing.T, input []interface{}, expected *imap.MailboxInfo) {
	output := &imap.MailboxInfo{}
	if err := output.Parse(input); err != nil {
		t.Fatal(err)
	}

	if fmt.Sprint(output.Attributes) != fmt.Sprint(expected.Attributes) {
		t.Fatal("Invalid flags:", output.Attributes)
	}
	if output.Delimiter != expected.Delimiter {
		t.Fatal("Invalid delimiter:", output.Delimiter)
	}
	if output.Name != expected.Name {
		t.Fatal("Invalid name:", output.Name)
	}
}

func testMailboxInfo_Format(t *testing.T, input *imap.MailboxInfo, expected []interface{}) {
	output := input.Format()

	if fmt.Sprint(output) != fmt.Sprint(expected) {
		t.Fatal("Invalid output:", output)
	}
}

var mailboxStatusTests = [...]struct{
	fields []interface{}
	status *imap.MailboxStatus
}{
	{
		fields: []interface{}{
			"MESSAGES", uint32(42),
			"RECENT", uint32(1),
			"UNSEEN", uint32(6),
			"UIDNEXT", uint32(65536),
			"UIDVALIDITY", uint32(4242),
		},
		status: &imap.MailboxStatus{
			Items: map[string]interface{}{
				"MESSAGES": nil,
				"RECENT": nil,
				"UNSEEN": nil,
				"UIDNEXT": nil,
				"UIDVALIDITY": nil,
			},
			Messages: 42,
			Recent: 1,
			Unseen: 6,
			UidNext: 65536,
			UidValidity: 4242,
		},
	},
}

func TestMailboxStatus_Format(t *testing.T) {
	for i, test := range mailboxStatusTests {
		fields := test.status.Format()
		sort.Sort(internal.MapListSorter(fields))

		sort.Sort(internal.MapListSorter(test.fields))

		if !reflect.DeepEqual(fields, test.fields) {
			t.Errorf("Invalid mailbox status fields for #%v: got \n%+v\n but expected \n%+v", i, fields, test.fields)
		}
	}
}
