package imapserver_test

import (
	"testing"

	"github.com/opsxolc/go-imap/v2/imapserver"
)

var matchListTests = []struct {
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

func TestMatchList(t *testing.T) {
	delim := '/'
	for _, test := range matchListTests {
		result := imapserver.MatchList(test.name, delim, test.ref, test.pattern)
		if result != test.result {
			t.Errorf("matching name %q with pattern %q and reference %q returns %v, but expected %v", test.name, test.pattern, test.ref, result, test.result)
		}
	}
}
