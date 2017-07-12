package utf7_test

import (
	"testing"

	"github.com/emersion/go-imap/utf7"
)

var decode = []struct {
	in  string
	out string
	ok  bool
}{
	// Basics (the inverse test on encode checks other valid inputs)
	{"", "", true},
	{"abc", "abc", true},
	{"&-abc", "&abc", true},
	{"abc&-", "abc&", true},
	{"a&-b&-c", "a&b&c", true},
	{"&ABk-", "\x19", true},
	{"&AB8-", "\x1F", true},
	{"ABk-", "ABk-", true},
	{"&-,&-&AP8-&-", "&,&\u00FF&", true},
	{"&-&-,&AP8-&-", "&&,\u00FF&", true},
	{"abc &- &AP8A,wD,- &- xyz", "abc & \u00FF\u00FF\u00FF & xyz", true},

	// Illegal code point in ASCII
	{"\x00", "", false},
	{"\x1F", "", false},
	{"abc\n", "", false},
	{"abc\x7Fxyz", "", false},
	{"\uFFFD", "", false},
	{"\u041C", "", false},

	// Invalid Base64 alphabet
	{"&/+8-", "", false},
	{"&*-", "", false},
	{"&ZeVnLIqe -", "", false},

	// CR and LF in Base64
	{"&ZeVnLIqe\r\n-", "", false},
	{"&ZeVnLIqe\r\n\r\n-", "", false},
	{"&ZeVn\r\n\r\nLIqe-", "", false},

	// Padding not stripped
	{"&AAAAHw=-", "", false},
	{"&AAAAHw==-", "", false},
	{"&AAAAHwB,AIA=-", "", false},
	{"&AAAAHwB,AIA==-", "", false},

	// One byte short
	{"&2A-", "", false},
	{"&2ADc-", "", false},
	{"&AAAAHwB,A-", "", false},
	{"&AAAAHwB,A=-", "", false},
	{"&AAAAHwB,A==-", "", false},
	{"&AAAAHwB,A===-", "", false},
	{"&AAAAHwB,AI-", "", false},
	{"&AAAAHwB,AI=-", "", false},
	{"&AAAAHwB,AI==-", "", false},

	// Implicit shift
	{"&", "", false},
	{"&Jjo", "", false},
	{"Jjo&", "", false},
	{"&Jjo&", "", false},
	{"&Jjo!", "", false},
	{"&Jjo+", "", false},
	{"abc&Jjo", "", false},

	// Null shift
	{"&AGE-&Jjo-", "", false},
	{"&U,BTFw-&ZeVnLIqe-", "", false},

	// ASCII in Base64
	{"&AGE-", "", false},            // "a"
	{"&ACY-", "", false},            // "&"
	{"&AGgAZQBsAGwAbw-", "", false}, // "hello"
	{"&JjoAIQ-", "", false},         // "\u263a!"

	// Bad surrogate
	{"&2AA-", "", false},    // U+D800
	{"&2AD-", "", false},    // U+D800
	{"&3AA-", "", false},    // U+DC00
	{"&2AAAQQ-", "", false}, // U+D800 'A'
	{"&2AD,,w-", "", false}, // U+D800 U+FFFF
	{"&3ADYAA-", "", false}, // U+DC00 U+D800
}

func TestDecoder(t *testing.T) {
	dec := utf7.Encoding.NewDecoder()

	for _, test := range decode {
		out, err := dec.String(test.in)
		if out != test.out {
			t.Errorf("UTF7Decode(%+q) expected %+q; got %+q", test.in, test.out, out)
		}
		if test.ok {
			if err != nil {
				t.Errorf("UTF7Decode(%+q) unexpected error; %v", test.in, err)
			}
		} else if err == nil {
			t.Errorf("UTF7Decode(%+q) expected error", test.in)
		}
	}
}
