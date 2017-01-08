package backendutil

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

func matchString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Match returns true if a message matches the provided criteria. Flag, sequence
// number and UID contrainsts are not checked.
func Match(e *message.Entity, c *imap.SearchCriteria) (bool, error) {
	// TODO: support encoded header fields for Bcc, Cc, From, To
	// TODO: add header size for Larger and Smaller

	h := mail.Header{e.Header}

	// TODO: optimize this
	b, err := ioutil.ReadAll(e.Body)
	if err != nil {
		return false, err
	}
	s := string(b)
	br := bytes.NewReader(b)
	e.Body = br

	if c.Bcc != "" && !matchString(h.Get("Bcc"), c.Bcc) {
		return false, nil
	}
	if !c.Before.IsZero() {
		if t, err := h.Date(); err != nil {
			return false, err
		} else if !t.Before(c.Before) {
			return false, nil
		}
	}
	if c.Body != "" && !matchString(s, c.Body) {
		return false, nil
	}
	if c.Cc != "" && !matchString(h.Get("Cc"), c.Cc) {
		return false, nil
	}
	if c.From != "" && !matchString(h.Get("From"), c.From) {
		return false, nil
	}
	if c.Header[0] != "" {
		key, value := c.Header[0], c.Header[1]
		values, ok := e.Header[key]
		if value == "" && !ok {
			return false, nil
		}
		if value != "" {
			ok := false
			for _, v := range values {
				if matchString(v, value) {
					ok = true
					break
				}
			}
			if !ok {
				return false, nil
			}
		}
	}
	if c.Larger > 0 && uint32(len(b)) < c.Larger {
		return false, nil
	}
	if c.Not != nil {
		ok, err := Match(e, c.Not)
		br.Seek(0, io.SeekStart)
		if err != nil || ok {
			return false, err
		}
	}
	if !c.On.IsZero() {
		if t, err := h.Date(); err != nil {
			return false, err
		} else if !t.Round(24 * time.Hour).Equal(c.On) {
			return false, nil
		}
	}
	if c.Or[0] != nil && c.Or[1] != nil {
		ok1, err := Match(e, c.Or[0])
		br.Seek(0, io.SeekStart)
		if err != nil {
			return ok1, err
		}

		ok2, err := Match(e, c.Or[1])
		br.Seek(0, io.SeekStart)
		if err != nil || (!ok1 && !ok2) {
			return false, err
		}
	}
	if !c.Since.IsZero() {
		if t, err := h.Date(); err != nil {
			return false, err
		} else if !t.After(c.Since) {
			return false, nil
		}
	}
	if c.Smaller > 0 && uint32(len(b)) > c.Smaller {
		return false, nil
	}
	if c.Subject != "" {
		if subject, err := h.Subject(); err != nil {
			return false, err
		} else if !matchString(subject, c.Subject) {
			return false, nil
		}
	}
	if c.Text != "" && !matchString(s, c.Text) {
		return false, nil // TODO: also search in header fields
	}
	if c.To != "" && !matchString(h.Get("To"), c.To) {
		return false, nil
	}

	return true, nil
}

func criteriaFlags(c *imap.SearchCriteria) (want, dontWant []string) {
	if c.Answered {
		want = append(want, imap.AnsweredFlag)
	}
	if c.Deleted {
		want = append(want, imap.DeletedFlag)
	}
	if c.Draft {
		want = append(want, imap.DraftFlag)
	}
	if c.Flagged {
		want = append(want, imap.FlaggedFlag)
	}
	if c.Keyword != "" {
		want = append(want, c.Keyword)
	}
	if c.New {
		want = append(want, imap.RecentFlag)
		dontWant = append(dontWant, imap.SeenFlag)
	}
	if c.Old {
		dontWant = append(dontWant, imap.RecentFlag)
	}
	if c.Recent {
		want = append(want, imap.RecentFlag)
	}
	if c.Seen {
		want = append(want, imap.SeenFlag)
	}
	if c.Unanswered {
		dontWant = append(dontWant, imap.AnsweredFlag)
	}
	if c.Undeleted {
		dontWant = append(dontWant, imap.DeletedFlag)
	}
	if c.Undraft {
		dontWant = append(dontWant, imap.DraftFlag)
	}
	if c.Unflagged {
		dontWant = append(dontWant, imap.FlaggedFlag)
	}
	if c.Unkeyword != "" {
		dontWant = append(dontWant, c.Unkeyword)
	}
	if c.Unseen {
		dontWant = append(dontWant, imap.SeenFlag)
	}
	return
}

func matchFlags(flags map[string]bool, c *imap.SearchCriteria) bool {
	want, dontWant := criteriaFlags(c)
	for _, f := range want {
		if !flags[f] {
			return false
		}
	}
	for _, f := range dontWant {
		if flags[f] {
			return false
		}
	}

	if c.Not != nil {
		if matchFlags(flags, c.Not) {
			return false
		}
	}

	if c.Or[0] != nil && c.Or[1] != nil {
		if !matchFlags(flags, c.Or[0]) && !matchFlags(flags, c.Or[1]) {
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
