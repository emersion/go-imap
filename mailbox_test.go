package imap_test

import (
	"fmt"
	"testing"

	"github.com/emersion/go-imap"
)

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
