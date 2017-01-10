package backendutil

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

func matchString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func bufferBody(e *message.Entity) (*bytes.Buffer, error) {
	b := new(bytes.Buffer)
	if _, err := io.Copy(b, e.Body); err != nil {
		return nil, err
	}
	e.Body = b
	return b, nil
}

func matchBody(e *message.Entity, substr string) (bool, error) {
	if s, ok := e.Body.(fmt.Stringer); ok {
		return matchString(s.String(), substr), nil
	}

	b, err := bufferBody(e)
	if err != nil {
		return false, err
	}
	return matchString(b.String(), substr), nil
}

type lengther interface {
	Len() int
}

func bodyLen(e *message.Entity) (int, error) {
	if l, ok := e.Body.(lengther); ok {
		return l.Len(), nil
	}

	b, err := bufferBody(e)
	if err != nil {
		return 0, err
	}
	return b.Len(), nil
}

// Match returns true if a message matches the provided criteria. Sequence
// number, UID, flag and internal date contrainsts are not checked.
func Match(e *message.Entity, c *imap.SearchCriteria) (bool, error) {
	// TODO: support encoded header fields for Bcc, Cc, From, To
	// TODO: add header size for Larger and Smaller

	h := mail.Header{e.Header}

	if !c.SentBefore.IsZero() || !c.SentSince.IsZero() {
		t, err := h.Date()
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
		values, ok := e.Header[key]
		for _, wantValue := range wantValues {
			if wantValue == "" && !ok {
				return false, nil
			}
			if wantValue != "" {
				ok := false
				for _, v := range values {
					if matchString(v, wantValue) {
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
	for _, body := range c.Body {
		if ok, err := matchBody(e, body); err != nil || !ok {
			return false, err
		}
	}
	for _, text := range c.Text {
		// TODO: also match header fields
		if ok, err := matchBody(e, text); err != nil || !ok {
			return false, err
		}
	}

	if c.Larger > 0 || c.Smaller > 0 {
		n, err := bodyLen(e)
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

	for _, not := range c.Not {
		ok, err := Match(e, not)
		if err != nil || ok {
			return false, err
		}
	}
	for _, or := range c.Or {
		ok1, err := Match(e, or[0])
		if err != nil {
			return ok1, err
		}

		ok2, err := Match(e, or[1])
		if err != nil || (!ok1 && !ok2) {
			return false, err
		}
	}

	return true, nil
}

func matchFlags(flags map[string]bool, c *imap.SearchCriteria) bool {
	for _, f := range c.WithFlags {
		if !flags[f] {
			return false
		}
	}
	for _, f := range c.WithoutFlags {
		if flags[f] {
			return false
		}
	}

	for _, not := range c.Not {
		if matchFlags(flags, not) {
			return false
		}
	}
	for _, or := range c.Or {
		if !matchFlags(flags, or[0]) && !matchFlags(flags, or[1]) {
			return false
		}
	}
	return true
}

// MatchFlags returns true if a flag list matches the provided criteria.
func MatchFlags(flags []string, c *imap.SearchCriteria) bool {
	flagsMap := make(map[string]bool)
	for _, f := range flags {
		flagsMap[f] = true
	}

	return matchFlags(flagsMap, c)
}

// MatchSeqNumAndUid returns true if a sequence number and a UID matches the
// provided criteria.
func MatchSeqNumAndUid(seqNum uint32, uid uint32, c *imap.SearchCriteria) bool {
	if c.SeqNum != nil && !c.SeqNum.Contains(seqNum) {
		return false
	}
	if c.Uid != nil && !c.Uid.Contains(uid) {
		return false
	}

	for _, not := range c.Not {
		if MatchSeqNumAndUid(seqNum, uid, not) {
			return false
		}
	}
	for _, or := range c.Or {
		if !MatchSeqNumAndUid(seqNum, uid, or[0]) && !MatchSeqNumAndUid(seqNum, uid, or[1]) {
			return false
		}
	}
	return true
}

// MatchDate returns true if a date matches the provided criteria.
func MatchDate(date time.Time, c *imap.SearchCriteria) bool {
	date = date.Round(24 * time.Hour)
	if !c.Since.IsZero() && !date.After(c.Since) {
		return false
	}
	if !c.Before.IsZero() && !date.Before(c.Before) {
		return false
	}

	for _, not := range c.Not {
		if MatchDate(date, not) {
			return false
		}
	}
	for _, or := range c.Or {
		if !MatchDate(date, or[0]) && !MatchDate(date, or[1]) {
			return false
		}
	}
	return true
}
