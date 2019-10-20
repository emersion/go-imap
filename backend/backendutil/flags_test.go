package backendutil

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap"
)

var updateFlagsTests = []struct {
	op    imap.FlagsOp
	flags []string
	res   []string
}{
	{
		op:    imap.AddFlags,
		flags: []string{"d", "e"},
		res:   []string{"a", "b", "c", "d", "e"},
	},
	{
		op:    imap.AddFlags,
		flags: []string{"a", "d", "b"},
		res:   []string{"a", "b", "c", "d"},
	},
	{
		op:    imap.RemoveFlags,
		flags: []string{"b", "v", "e", "a"},
		res:   []string{"c"},
	},
	{
		op:    imap.SetFlags,
		flags: []string{"a", "d", "e"},
		res:   []string{"a", "d", "e"},
	},
	// Test unknown op for code coverage.
	{
		op:    imap.FlagsOp("TestUnknownOp"),
		flags: []string{"a", "d", "e"},
		res:   []string{"a", "b", "c"},
	},
}

func TestUpdateFlags(t *testing.T) {
	flagsList := []string{"a", "b", "c"}
	for _, test := range updateFlagsTests {
		// Make a backup copy of 'test.flags'
		origFlags := append(test.flags[:0:0], test.flags...)
		// Copy flags
		current := append(flagsList[:0:0], flagsList...)
		got := UpdateFlags(current, test.op, test.flags)

		if !reflect.DeepEqual(got, test.res) {
			t.Errorf("Expected result to be \n%v\n but got \n%v", test.res, got)
		}
		// Verify that 'test.flags' wasn't modified
		if !reflect.DeepEqual(origFlags, test.flags) {
			t.Errorf("Unexpected change to operation flags list changed \nbefore %v\n after \n%v",
				origFlags, test.flags)
		}
	}
}

func TestUpdateFlags_Recent(t *testing.T) {
	current := []string{}

	current = UpdateFlags(current, imap.SetFlags, []string{imap.RecentFlag})

	res := []string{imap.RecentFlag}
	if !reflect.DeepEqual(current, res) {
		t.Errorf("Expected result to be \n%v\n but got \n%v", res, current)
	}

	current = UpdateFlags(current, imap.SetFlags, []string{"something"})

	res = []string{imap.RecentFlag, "something"}
	if !reflect.DeepEqual(current, res) {
		t.Errorf("Expected result to be \n%v\n but got \n%v", res, current)
	}

	current = UpdateFlags(current, imap.SetFlags, []string{"another", imap.RecentFlag})

	res = []string{imap.RecentFlag, "another"}
	if !reflect.DeepEqual(current, res) {
		t.Errorf("Expected result to be \n%v\n but got \n%v", res, current)
	}
}
