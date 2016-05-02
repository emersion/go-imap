package common

import (
	"testing"
)

func TestLiteral(t *testing.T) {
	literal := NewLiteral([]byte("hello world"))

	if string(literal.Bytes()) != "hello world" {
		t.Error("Invalid literal bytes")
	}
	if literal.String() != "hello world" {
		t.Error("Invalid literal string")
	}
	if literal.Len() != 11 {
		t.Error("Invalid literal length")
	}

	if literal.field() != "{11}" {
		t.Error("Invalid literal field")
	}
}
