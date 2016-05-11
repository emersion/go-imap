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
		t.Fatal(err)
	}
	if b.String() != "A001 NOOP\r\n" {
		t.Fatal("Not the expected command")
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
		t.Fatal(err)
	}
	if b.String() != "A002 LOGIN username password\r\n" {
		t.Fatal("Not the expected command")
	}
}

func TestCommand_Parse_NoArgs(t *testing.T) {
	fields := []interface{}{"a", "NOOP"}

	cmd := &common.Command{}

	if err := cmd.Parse(fields); err != nil {
		t.Fatal(err)
	}
	if cmd.Tag != "a" {
		t.Error("Invalid tag:", cmd.Tag)
	}
	if cmd.Name != "NOOP" {
		t.Error("Invalid name:", cmd.Name)
	}
	if len(cmd.Arguments) != 0 {
		t.Error("Invalid arguments:", cmd.Arguments)
	}
}

func TestCommand_Parse_WithArgs(t *testing.T) {
	fields := []interface{}{"a", "LOGIN", "username", "password"}

	cmd := &common.Command{}

	if err := cmd.Parse(fields); err != nil {
		t.Fatal(err)
	}
	if cmd.Tag != "a" {
		t.Error("Invalid tag:", cmd.Tag)
	}
	if cmd.Name != "LOGIN" {
		t.Error("Invalid name:", cmd.Name)
	}
	if len(cmd.Arguments) != 2 {
		t.Error("Invalid arguments:", cmd.Arguments)
	}
	if username, ok := cmd.Arguments[0].(string); !ok || username != "username" {
		t.Error("Invalid first argument:", cmd.Arguments[0])
	}
	if password, ok := cmd.Arguments[1].(string); !ok || password != "password" {
		t.Error("Invalid second argument:", cmd.Arguments[1])
	}
}
