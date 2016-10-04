package imap

import (
	"fmt"
	"strings"
	"time"
)

// Date and time layouts.
const (
	// Described in RFC 1730 on page 55.
	DateLayout = "2-Jan-2006"
	// Described in RFC 1730 on page 55.
	DateTimeLayout = "2-Jan-2006 15:04:05 -0700"
	// Described in RFC 5322 section 3.3.
	MessageDateTimeLayout = "Mon, 02 Jan 2006 15:04:05 -0700"
)

// time.Time with a specific layout.
type Date time.Time
type DateTime time.Time
type MessageDateTime time.Time

// Permutations of the layouts defined in RFC 5322, section 3.3.
var messageDateTimeLayouts = [...]string{
	MessageDateTimeLayout, // popular, try it first
	"2 Jan 2006 15:04:05 -0700",
	"2 Jan 2006 15:04:05 MST",
	"2 Jan 2006 15:04:05 -0700 (MST)",
	"2 Jan 2006 15:04 -0700",
	"2 Jan 2006 15:04 MST",
	"2 Jan 2006 15:04 -0700 (MST)",
	"2 Jan 06 15:04:05 -0700",
	"2 Jan 06 15:04:05 MST",
	"2 Jan 06 15:04:05 -0700 (MST)",
	"2 Jan 06 15:04 -0700",
	"2 Jan 06 15:04 MST",
	"2 Jan 06 15:04 -0700 (MST)",
	"02 Jan 2006 15:04:05 -0700",
	"02 Jan 2006 15:04:05 MST",
	"02 Jan 2006 15:04:05 -0700 (MST)",
	"02 Jan 2006 15:04 -0700",
	"02 Jan 2006 15:04 MST",
	"02 Jan 2006 15:04 -0700 (MST)",
	"02 Jan 06 15:04:05 -0700",
	"02 Jan 06 15:04:05 MST",
	"02 Jan 06 15:04:05 -0700 (MST)",
	"02 Jan 06 15:04 -0700",
	"02 Jan 06 15:04 MST",
	"02 Jan 06 15:04 -0700 (MST)",
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
	"Mon, 2 Jan 2006 15:04 -0700",
	"Mon, 2 Jan 2006 15:04 MST",
	"Mon, 2 Jan 2006 15:04 -0700 (MST)",
	"Mon, 2 Jan 06 15:04:05 -0700",
	"Mon, 2 Jan 06 15:04:05 MST",
	"Mon, 2 Jan 06 15:04:05 -0700 (MST)",
	"Mon, 2 Jan 06 15:04 -0700",
	"Mon, 2 Jan 06 15:04 MST",
	"Mon, 2 Jan 06 15:04 -0700 (MST)",
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05 -0700 (MST)",
	"Mon, 02 Jan 2006 15:04 -0700",
	"Mon, 02 Jan 2006 15:04 MST",
	"Mon, 02 Jan 2006 15:04 -0700 (MST)",
	"Mon, 02 Jan 06 15:04:05 -0700",
	"Mon, 02 Jan 06 15:04:05 MST",
	"Mon, 02 Jan 06 15:04:05 -0700 (MST)",
	"Mon, 02 Jan 06 15:04 -0700",
	"Mon, 02 Jan 06 15:04 MST",
	"Mon, 02 Jan 06 15:04 -0700 (MST)",
}

// Try parsing the date based on the layouts defined in RFC 5322, section 3.3.
// Inspired by https://github.com/golang/go/blob/master/src/net/mail/message.go
func ParseMessageDateTime(maybeDate string) (time.Time, error) {
	maybeDate = strings.TrimSpace(maybeDate)
	for _, layout := range messageDateTimeLayouts {
		parsed, err := time.Parse(layout, maybeDate)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("date %s could not be parsed", maybeDate)
}

// Try parsing an IMAP date with time.
func ParseDateTime(maybeDate string) (time.Time, error) {
	maybeDate = strings.TrimSpace(maybeDate)
	parsed, err := time.Parse(DateTimeLayout, maybeDate)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("date %s could not be parsed", maybeDate)
}

// Try parsing an IMAP date.
func ParseDate(maybeDate string) (time.Time, error) {
	maybeDate = strings.TrimSpace(maybeDate)
	parsed, err := time.Parse(DateLayout, maybeDate)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("date %s could not be parsed", maybeDate)
}
