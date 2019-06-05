package imap_test

import (
	"fmt"
	"reflect"
	"sort"
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

var mailboxInfoTests = []struct {
	fields []interface{}
	info   *imap.MailboxInfo
}{
	{
		fields: []interface{}{
			[]interface{}{"\\Noselect", "\\Recent", "\\Unseen"},
			"/",
			"INBOX",
		},
		info: &imap.MailboxInfo{
			Attributes: []string{"\\Noselect", "\\Recent", "\\Unseen"},
			Delimiter:  "/",
			Name:       "INBOX",
		},
	},
}

func TestMailboxInfo_Parse(t *testing.T) {
	for _, test := range mailboxInfoTests {
		info := &imap.MailboxInfo{}
		if err := info.Parse(test.fields); err != nil {
			t.Fatal(err)
		}

		if fmt.Sprint(info.Attributes) != fmt.Sprint(test.info.Attributes) {
			t.Fatal("Invalid flags:", info.Attributes)
		}
		if info.Delimiter != test.info.Delimiter {
			t.Fatal("Invalid delimiter:", info.Delimiter)
		}
		if info.Name != test.info.Name {
			t.Fatal("Invalid name:", info.Name)
		}
	}
}

func TestMailboxInfo_Format(t *testing.T) {
	for _, test := range mailboxInfoTests {
		fields := test.info.Format()

		if fmt.Sprint(fields) != fmt.Sprint(test.fields) {
			t.Fatal("Invalid fields:", fields)
		}
	}
}

var mailboxInfoMatchTests = []struct {
	name, ref, pattern string
	result             bool
}{
	{name: "INBOX", pattern: "INBOX", result: true},
	{name: "INBOX", pattern: "Asuka", result: false},
	{name: "INBOX", pattern: "*", result: true},
	{name: "INBOX", pattern: "%", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "*", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "%", result: false},
	{name: "Neon Genesis Evangelion/Misato", pattern: "Neon Genesis Evangelion/*", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "Neon Genesis Evangelion/%", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "Neo* Evangelion/Misato", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "Neo% Evangelion/Misato", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "*Eva*/Misato", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "%Eva%/Misato", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "*X*/Misato", result: false},
	{name: "Neon Genesis Evangelion/Misato", pattern: "%X%/Misato", result: false},
	{name: "Neon Genesis Evangelion/Misato", pattern: "Neon Genesis Evangelion/Mi%o", result: true},
	{name: "Neon Genesis Evangelion/Misato", pattern: "Neon Genesis Evangelion/Mi%too", result: false},
	{name: "Misato/Misato", pattern: "Mis*to/Misato", result: true},
	{name: "Misato/Misato", pattern: "Mis*to", result: true},
	{name: "Misato/Misato/Misato", pattern: "Mis*to/Mis%to", result: true},
	{name: "Misato/Misato", pattern: "Mis**to/Misato", result: true},
	{name: "Misato/Misato", pattern: "Misat%/Misato", result: true},
	{name: "Misato/Misato", pattern: "Misat%Misato", result: false},
	{name: "Misato/Misato", ref: "Misato", pattern: "Misato", result: true},
	{name: "Misato/Misato", ref: "Misato/", pattern: "Misato", result: true},
	{name: "Misato/Misato", ref: "Shinji", pattern: "/Misato/*", result: true},
	{name: "Misato/Misato", ref: "Misato", pattern: "/Misato", result: false},
	{name: "Misato/Misato", ref: "Misato", pattern: "Shinji", result: false},
	{name: "Misato/Misato", ref: "Shinji", pattern: "Misato", result: false},
}

func TestMailboxInfo_Match(t *testing.T) {
	for _, test := range mailboxInfoMatchTests {
		info := &imap.MailboxInfo{Name: test.name, Delimiter: "/"}
		result := info.Match(test.ref, test.pattern)
		if result != test.result {
			t.Errorf("Matching name %q with pattern %q and reference %q returns %v, but expected %v", test.name, test.pattern, test.ref, result, test.result)
		}
	}
}

func TestNewMailboxStatus(t *testing.T) {
	status := imap.NewMailboxStatus("INBOX", []imap.StatusItem{imap.StatusMessages, imap.StatusUnseen})

	expected := &imap.MailboxStatus{
		Name:  "INBOX",
		Items: map[imap.StatusItem]interface{}{imap.StatusMessages: nil, imap.StatusUnseen: nil},
	}

	if !reflect.DeepEqual(expected, status) {
		t.Errorf("Invalid mailbox status: expected \n%+v\n but got \n%+v", expected, status)
	}
}

var mailboxStatusTests = [...]struct {
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
			Items: map[imap.StatusItem]interface{}{
				imap.StatusMessages:    nil,
				imap.StatusRecent:      nil,
				imap.StatusUnseen:      nil,
				imap.StatusUidNext:     nil,
				imap.StatusUidValidity: nil,
			},
			Messages:    42,
			Recent:      1,
			Unseen:      6,
			UidNext:     65536,
			UidValidity: 4242,
		},
	},
}

func TestMailboxStatus_Parse(t *testing.T) {
	for i, test := range mailboxStatusTests {
		status := &imap.MailboxStatus{}
		if err := status.Parse(test.fields); err != nil {
			t.Errorf("Expected no error while parsing mailbox status #%v, got: %v", i, err)
			continue
		}

		if !reflect.DeepEqual(status, test.status) {
			t.Errorf("Invalid parsed mailbox status for #%v: got \n%+v\n but expected \n%+v", i, status, test.status)
		}
	}
}

func TestMailboxStatus_Format(t *testing.T) {
	for i, test := range mailboxStatusTests {
		fields := test.status.Format()

		// MapListSorter does not know about RawString and will panic.
		stringFields := make([]interface{}, 0, len(fields))
		for _, field := range fields {
			if s, ok := field.(imap.RawString); ok {
				stringFields = append(stringFields, string(s))
			} else {
				stringFields = append(stringFields, field)
			}
		}

		sort.Sort(internal.MapListSorter(stringFields))

		sort.Sort(internal.MapListSorter(test.fields))

		if !reflect.DeepEqual(stringFields, test.fields) {
			t.Errorf("Invalid mailbox status fields for #%v: got \n%+v\n but expected \n%+v", i, fields, test.fields)
		}
	}
}
