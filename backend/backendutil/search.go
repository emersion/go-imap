package backendutil

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/textproto"
)

func matchString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func bufferBody(body *io.Reader) (*bytes.Buffer, error) {
	b := new(bytes.Buffer)
	if _, err := io.Copy(b, *body); err != nil {
		return nil, err
	}
	*body = b
	return b, nil
}

func matchBody(body *io.Reader, substr string) (bool, error) {
	if s, ok := (*body).(fmt.Stringer); ok {
		return matchString(s.String(), substr), nil
	}

	b, err := bufferBody(body)
	if err != nil {
		return false, err
	}
	return matchString(b.String(), substr), nil
}

type lengther interface {
	Len() int
}

func bodyLen(body *io.Reader) (int, error) {
	if l, ok := (*body).(lengther); ok {
		return l.Len(), nil
	}

	b, err := bufferBody(body)
	if err != nil {
		return 0, err
	}
	return b.Len(), nil
}

// Match returns true if a message and its metadata matches the provided
// criteria.
func Match(hdr textproto.Header, body io.Reader, seqNum, uid uint32, date time.Time, flags []string, c *imap.SearchCriteria) (bool, error) {
	return match(hdr, &body, seqNum, uid, date, flags, c)
}

func match(hdr textproto.Header, body *io.Reader, seqNum, uid uint32, date time.Time, flags []string, c *imap.SearchCriteria) (bool, error) {
	// TODO: support encoded header fields for Bcc, Cc, From, To
	// TODO: add header size for Larger and Smaller

	if !c.SentBefore.IsZero() || !c.SentSince.IsZero() {
		t, err := mail.ParseDate(hdr.Get("Date"))
		if err != nil {
			return false, err
		}
		t = t.Round(24 * time.Hour)

		if !c.SentBefore.IsZero() && !t.Before(c.SentBefore) {
			return false, nil
		}
		if !c.SentSince.IsZero() && !t.After(c.SentSince) {
			return false, nil
		}
	}

	for key, wantValues := range c.Header {
		ok := hdr.Has(key)
		for _, wantValue := range wantValues {
			if wantValue == "" && !ok {
				return false, nil
			}
			if wantValue != "" {
				ok := false
				values := hdr.FieldsByKey(key)
				for values.Next() {
					if matchString(values.Value(), wantValue) {
						ok = true
						break
					}
				}
				if !ok {
					return false, nil
				}
			}
		}
	}
	for _, searchBody := range c.Body {
		if ok, err := matchBody(body, searchBody); err != nil || !ok {
			return false, err
		}
	}
	for _, text := range c.Text {
		// TODO: also match header fields
		if ok, err := matchBody(body, text); err != nil || !ok {
			return false, err
		}
	}

	if c.Larger > 0 || c.Smaller > 0 {
		n, err := bodyLen(body)
		if err != nil {
			return false, err
		}

		if c.Larger > 0 && uint32(n) < c.Larger {
			return false, nil
		}
		if c.Smaller > 0 && uint32(n) > c.Smaller {
			return false, nil
		}
	}

	if !c.Since.IsZero() || !c.Before.IsZero() {
		if !matchDate(date, c) {
			return false, nil
		}
	}

	if c.WithFlags != nil || c.WithoutFlags != nil {
		if !matchFlags(flags, c) {
			return false, nil
		}
	}

	if c.SeqNum != nil || c.Uid != nil {
		if !matchSeqNumAndUid(seqNum, uid, c) {
			return false, nil
		}
	}

	for _, not := range c.Not {
		ok, err := match(hdr, body, seqNum, uid, date, flags, not)
		if err != nil || ok {
			return false, err
		}
	}
	for _, or := range c.Or {
		ok1, err := match(hdr, body, seqNum, uid, date, flags, or[0])
		if err != nil {
			return ok1, err
		}

		ok2, err := match(hdr, body, seqNum, uid, date, flags, or[1])
		if err != nil || (!ok1 && !ok2) {
			return false, err
		}
	}

	return true, nil
}

func matchFlags(flags []string, c *imap.SearchCriteria) bool {
	flagsMap := make(map[string]bool)
	for _, f := range flags {
		flagsMap[f] = true
	}

	for _, f := range c.WithFlags {
		if !flagsMap[f] {
			return false
		}
	}
	for _, f := range c.WithoutFlags {
		if flagsMap[f] {
			return false
		}
	}

	return true
}

func matchSeqNumAndUid(seqNum uint32, uid uint32, c *imap.SearchCriteria) bool {
	if c.SeqNum != nil && !c.SeqNum.Contains(seqNum) {
		return false
	}
	if c.Uid != nil && !c.Uid.Contains(uid) {
		return false
	}
	return true
}

func matchDate(date time.Time, c *imap.SearchCriteria) bool {
	date = date.Round(24 * time.Hour)
	if !c.Since.IsZero() && !date.After(c.Since) {
		return false
	}
	if !c.Before.IsZero() && !date.Before(c.Before) {
		return false
	}
	return true
}
