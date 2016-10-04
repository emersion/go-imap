package imap

import (
	"fmt"
	"time"
)

// Date and time layouts.
// Dovecot adds a leading zero to dates:
//   https://github.com/dovecot/core/blob/4fbd5c5e113078e72f29465ccc96d44955ceadc2/src/lib-imap/imap-date.c#L166
// Cyrus adds a leading space to dates:
//   https://github.com/cyrusimap/cyrus-imapd/blob/1cb805a3bffbdf829df0964f3b802cdc917e76db/lib/times.c#L543
const (
	// Described in RFC 1730 on page 55.
	DateLayout = "_2-Jan-2006"
	// Described in RFC 1730 on page 55.
	DateTimeLayout = "_2-Jan-2006 15:04:05 -0700"
	// Described in RFC 5322 section 3.3.
	messageDateTimeLayout = "Mon, 02 Jan 2006 15:04:05 -0700"
)

// time.Time with a specific layout.
type (
	Date            time.Time
	DateTime        time.Time
	messageDateTime time.Time
)

// Permutations of the layouts defined in RFC 5322, section 3.3.
var messageDateTimeLayouts = [...]string{
	messageDateTimeLayout, // popular, try it first
	"_2 Jan 2006 15:04:05 -0700",
	"_2 Jan 2006 15:04:05 MST",
	"_2 Jan 2006 15:04:05 -0700 (MST)",
	"_2 Jan 2006 15:04 -0700",
	"_2 Jan 2006 15:04 MST",
	"_2 Jan 2006 15:04 -0700 (MST)",
	"_2 Jan 06 15:04:05 -0700",
	"_2 Jan 06 15:04:05 MST",
	"_2 Jan 06 15:04:05 -0700 (MST)",
	"_2 Jan 06 15:04 -0700",
	"_2 Jan 06 15:04 MST",
	"_2 Jan 06 15:04 -0700 (MST)",
	"Mon, _2 Jan 2006 15:04:05 -0700",
	"Mon, _2 Jan 2006 15:04:05 MST",
	"Mon, _2 Jan 2006 15:04:05 -0700 (MST)",
	"Mon, _2 Jan 2006 15:04 -0700",
	"Mon, _2 Jan 2006 15:04 MST",
	"Mon, _2 Jan 2006 15:04 -0700 (MST)",
	"Mon, _2 Jan 06 15:04:05 -0700",
	"Mon, _2 Jan 06 15:04:05 MST",
	"Mon, _2 Jan 06 15:04:05 -0700 (MST)",
	"Mon, _2 Jan 06 15:04 -0700",
	"Mon, _2 Jan 06 15:04 MST",
	"Mon, _2 Jan 06 15:04 -0700 (MST)",
}

// Try parsing the date based on the layouts defined in RFC 5322, section 3.3.
// Inspired by https://github.com/golang/go/blob/master/src/net/mail/message.go
func parseMessageDateTime(maybeDate string) (time.Time, error) {
	for _, layout := range messageDateTimeLayouts {
		parsed, err := time.Parse(layout, maybeDate)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("date %s could not be parsed", maybeDate)
}

// Try parsing an IMAP date with time.
func parseDateTime(maybeDate string) (time.Time, error) {
	parsed, err := time.Parse(DateTimeLayout, maybeDate)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("date %s could not be parsed", maybeDate)
}

// Try parsing an IMAP date.
func parseDate(maybeDate string) (time.Time, error) {
	parsed, err := time.Parse(DateLayout, maybeDate)
	if err == nil {
		return parsed, nil
	}

	return time.Time{}, fmt.Errorf("date %s could not be parsed", maybeDate)
}
