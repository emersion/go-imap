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
}

func TestUpdateFlags(t *testing.T) {
	current := []string{"a", "b", "c"}
	for _, test := range updateFlagsTests {
		got := UpdateFlags(current[:], test.op, test.flags)

		if !reflect.DeepEqual(got, test.res) {
			t.Errorf("Expected result to be \n%v\n but got \n%v", test.res, got)
		}
	}
}
