package common_test

import (
	"bytes"
	"testing"

	"github.com/emersion/imap/common"
)

func TestCommand_WriteTo_NoArgs(t *testing.T) {
	var b bytes.Buffer
	w := common.NewWriter(&b)

	cmd := &common.Command{
		Tag: "A001",
		Name: "NOOP",
	}

	if _, err := cmd.WriteTo(w); err != nil {
		t.Error(err)
	}
	if b.String() != "A001 NOOP\r\n" {
		t.Error("Not the excepted command")
	}
}

func TestCommand_WriteTo_WithArgs(t *testing.T) {
	var b bytes.Buffer
	w := common.NewWriter(&b)

	cmd := &common.Command{
		Tag: "A002",
		Name: "LOGIN",
		Arguments: []interface{}{"username", "password"},
	}

	if _, err := cmd.WriteTo(w); err != nil {
		t.Error(err)
	}
	if b.String() != "A002 LOGIN username password\r\n" {
		t.Error("Not the excepted command")
	}
}
