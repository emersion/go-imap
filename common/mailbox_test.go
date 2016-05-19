package common_test

import (
	"testing"
	"fmt"

	"github.com/emersion/go-imap/common"
)

func TestMailboxInfo(t *testing.T) {
	fields := []interface{}{
		[]interface{}{"\\Noselect", "\\Recent", "\\Unseen"},
		"/",
		"INBOX",
	}
	info := &common.MailboxInfo{
		Flags: []string{"\\Noselect", "\\Recent", "\\Unseen"},
		Delimiter: "/",
		Name: "INBOX",
	}

	testMailboxInfo_Parse(t, fields, info)
	testMailboxInfo_Format(t, info, fields)
}

func testMailboxInfo_Parse(t *testing.T, input []interface{}, expected *common.MailboxInfo) {
	output := &common.MailboxInfo{}
	if err := output.Parse(input); err != nil {
		t.Fatal(err)
	}

	if fmt.Sprint(output.Flags) != fmt.Sprint(expected.Flags) {
		t.Fatal("Invalid flags:", output.Flags)
	}
	if output.Delimiter != expected.Delimiter {
		t.Fatal("Invalid delimiter:", output.Delimiter)
	}
	if output.Name != expected.Name {
		t.Fatal("Invalid name:", output.Name)
	}
}

func testMailboxInfo_Format(t *testing.T, input *common.MailboxInfo, expected []interface{}) {
	output := input.Format()

	if fmt.Sprint(output) != fmt.Sprint(expected) {
		t.Fatal("Invalid output:", output)
	}
}
