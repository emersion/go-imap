package internal

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const (
	DateTimeLayout = "_2-Jan-2006 15:04:05 -0700"
	DateLayout     = "2-Jan-2006"
)

const StatusItemRecent imap.StatusItem = "RECENT" // removed in IMAP4rev2

// Fetch items removed in IMAP4rev2.
var (
	FetchItemRFC822       imap.FetchItem = imap.FetchItemKeyword("RFC822")        // equivalent to BODY[]
	FetchItemRFC822Header imap.FetchItem = imap.FetchItemKeyword("RFC822.HEADER") // equivalent to BODY.PEEK[HEADER]
	FetchItemRFC822Text   imap.FetchItem = imap.FetchItemKeyword("RFC822.TEXT")   // equivalent to BODY[TEXT]
)

const FlagRecent imap.Flag = "\\Recent" // removed in IMAP4rev2

func DecodeDateTime(dec *imapwire.Decoder) (time.Time, error) {
	var s string
	if !dec.Quoted(&s) {
		return time.Time{}, nil
	}
	t, err := time.Parse(DateTimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("in date-time: %v", err) // TODO: use imapwire.DecodeExpectError?
	}
	return t, err
}

func ExpectDateTime(dec *imapwire.Decoder) (time.Time, error) {
	t, err := DecodeDateTime(dec)
	if err != nil {
		return t, err
	}
	if !dec.Expect(!t.IsZero(), "date-time") {
		return t, dec.Err()
	}
	return t, nil
}

func ExpectDate(dec *imapwire.Decoder) (time.Time, error) {
	var s string
	if !dec.ExpectAString(&s) {
		return time.Time{}, dec.Err()
	}
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("in date: %v", err) // use imapwire.DecodeExpectError?
	}
	return t, nil
}

func ExpectFlagList(dec *imapwire.Decoder) ([]imap.Flag, error) {
	var flags []imap.Flag
	err := dec.ExpectList(func() error {
		flag, err := ExpectFlag(dec)
		if err != nil {
			return err
		}
		flags = append(flags, imap.Flag(flag))
		return nil
	})
	return flags, err
}

func ExpectFlag(dec *imapwire.Decoder) (string, error) {
	isSystem := dec.Special('\\')
	if isSystem && dec.Special('*') {
		return "\\*", nil // flag-perm
	}
	var name string
	if !dec.ExpectAtom(&name) {
		return "", fmt.Errorf("in flag: %w", dec.Err())
	}
	if isSystem {
		name = "\\" + name
	}
	return name, nil
}
