package common_test

import (
	"testing"
	"time"

	"github.com/emersion/imap/common"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		dateStr string
		exp     time.Time
	}{
		// RFC 5322, Appendix A.1.1
		{
			"Fri, 21 Nov 1997 09:55:06 -0600",
			time.Date(1997, 11, 21, 9, 55, 6, 0, time.FixedZone("", -6*60*60)),
		},
		// RFC5322, Appendix A.6.2
		// Obsolete date.
		{
			"21 Nov 97 09:55:06 GMT",
			time.Date(1997, 11, 21, 9, 55, 6, 0, time.FixedZone("GMT", 0)),
		},
		// Commonly found format not specified by RFC 5322.
		{
			"Fri, 21 Nov 1997 09:55:06 -0600 (MDT)",
			time.Date(1997, 11, 21, 9, 55, 6, 0, time.FixedZone("", -6*60*60)),
		},
	}
	for _, test := range tests {
		date, err := common.ParseDate(test.dateStr)
		if err != nil {
			t.Errorf("Failed parsing %q: %v", test.dateStr, err)
			continue
		}
		if !date.Equal(test.exp) {
			t.Errorf("Parse of %q: got %+v, want %+v", test.dateStr, date, test.exp)
		}
	}
}
