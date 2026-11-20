package imapserver

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestExtractPartialOverflow(t *testing.T) {
	b := []byte("hello world")
	for _, tc := range []struct {
		name    string
		partial imap.SectionPartial
		want    string
	}{
		{"overflow", imap.SectionPartial{Offset: 5, Size: 1<<63 - 1}, " world"},
		{"negative-size", imap.SectionPartial{Offset: 5, Size: -1}, " world"},
		{"normal", imap.SectionPartial{Offset: 2, Size: 3}, "llo"},
		{"clamped", imap.SectionPartial{Offset: 6, Size: 100}, "world"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(extractPartial(b, &tc.partial)); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
