package imap

import (
	"testing"
	"time"
)

var expectedDateTime = time.Date(2009, time.November, 2, 23, 0, 0, 0, time.FixedZone("", -6*60*60))
var expectedDate = time.Date(2009, time.November, 2, 0, 0, 0, 0, time.FixedZone("", 0))

func TestParseMessageDateTime(t *testing.T) {
	tests := []struct {
		in  string
		out time.Time
		ok  bool
	}{
		// some permutations
		{"2 Nov 2009 23:00 -0600", expectedDateTime, true},
		{"Tue, 2 Nov 2009 23:00:00 -0600", expectedDateTime, true},
		{"Tue, 2 Nov 2009 23:00:00 -0600 (MST)", expectedDateTime, true},

		// whitespace
		{" 2 Nov 2009 23:00 -0600", expectedDateTime, true},
		{"Tue,  2 Nov 2009 23:00:00 -0600", expectedDateTime, true},
		{"Tue,  2 Nov 2009 23:00:00 -0600 (MST)", expectedDateTime, true},

		// invalid
		{"abc10 Nov 2009 23:00 -0600123", expectedDateTime, false},
		{"10.Nov.2009 11:00:00 -9900", expectedDateTime, false},
	}
	for _, test := range tests {
		out, err := parseMessageDateTime(test.in)
		if !test.ok {
			if err == nil {
				t.Errorf("ParseMessageDateTime(%q) expected error; got %q", test.in, out)
			}
		} else if err != nil {
			t.Errorf("ParseMessageDateTime(%q) expected %q; got %v", test.in, test.out, err)
		} else if !out.Equal(test.out) {
			t.Errorf("ParseMessageDateTime(%q) expected %q; got %q", test.in, test.out, out)
		}
	}
}

func TestParseDateTime(t *testing.T) {
	tests := []struct {
		in  string
		out time.Time
		ok  bool
	}{
		{"2-Nov-2009 23:00:00 -0600", expectedDateTime, true},

		// whitespace
		{" 2-Nov-2009 23:00:00 -0600", expectedDateTime, true},

		// invalid or incorrect
		{"10-Nov-2009", time.Time{}, false},
		{"abc10-Nov-2009 23:00:00 -0600123", time.Time{}, false},
	}
	for _, test := range tests {
		out, err := time.Parse(DateTimeLayout, test.in)
		if !test.ok {
			if err == nil {
				t.Errorf("ParseDateTime(%q) expected error; got %q", test.in, out)
			}
		} else if err != nil {
			t.Errorf("ParseDateTime(%q) expected %q; got %v", test.in, test.out, err)
		} else if !out.Equal(test.out) {
			t.Errorf("ParseDateTime(%q) expected %q; got %q", test.in, test.out, out)
		}
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		in  string
		out time.Time
		ok  bool
	}{
		{"2-Nov-2009", expectedDate, true},
		{" 2-Nov-2009", expectedDate, true},
	}
	for _, test := range tests {
		out, err := time.Parse(DateLayout, test.in)
		if !test.ok {
			if err == nil {
				t.Errorf("ParseDate(%q) expected error; got %q", test.in, out)
			}
		} else if err != nil {
			t.Errorf("ParseDate(%q) expected %q; got %v", test.in, test.out, err)
		} else if !out.Equal(test.out) {
			t.Errorf("ParseDate(%q) expected %q; got %q", test.in, test.out, out)
		}
	}
}
