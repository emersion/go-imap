package imap

import (
	"errors"
	"strings"
	"time"
)

// TODO: support AND with same fields (e.g. BCC mickey BCC mouse)

// A search criteria.
// See RFC 3501 section 6.4.4 for a description of each field.
type SearchCriteria struct {
	SeqSet *SeqSet
	Answered bool
	Bcc string
	Before time.Time
	Body string
	Cc string
	Deleted bool
	Draft bool
	Flagged bool
	From string
	Header [2]string
	Keyword string
	Larger uint32
	New bool
	Not *SearchCriteria
	Old bool
	On time.Time
	Or [2]*SearchCriteria
	Recent bool
	Seen bool
	SentBefore time.Time
	SentOn time.Time
	SentSince time.Time
	Since time.Time
	Smaller uint32
	Subject string
	Text string
	To string
	Uid *SeqSet
	Unanswered bool
	Undeleted bool
	Undraft bool
	Unflagged bool
	Unkeyword string
	Unseen bool
}

// Parse search criteria from fields.
func (c *SearchCriteria) Parse(fields []interface{}) error {
	// TODO: do not panic when criteria is malformed

	for i := 0; i < len(fields); i++ {
		f, ok := fields[i].(string)
		if !ok {
			return errors.New("Invalid search criteria field")
		}

		switch strings.ToUpper(f) {
		case "ALL":
			// Nothing to do
		case "ANSWERED":
			c.Answered = true
		case "BCC":
			i++
			c.Bcc, _ = fields[i].(string)
		case "BEFORE":
			i++
			c.Before, _ = ParseDate(fields[i])
		case "BODY":
			i++
			c.Body, _ = fields[i].(string)
		case "CC":
			i++
			c.Cc, _ = fields[i].(string)
		case "DELETED":
			c.Deleted = true
		case "DRAFT":
			c.Draft = true
		case "FLAGGED":
			c.Flagged = true
		case "FROM":
			i++
			c.From, _ = fields[i].(string)
		case "HEADER":
			i++
			name, _ := fields[i].(string)

			i++
			value, _ := fields[i].(string)

			c.Header = [2]string{name, value}
		case "KEYWORD":
			i++
			c.Keyword, _ = fields[i].(string)
		case "LARGER":
			i++
			c.Larger, _ = ParseNumber(fields[i])
		case "NEW":
			c.New = true
		case "NOT":
			i++
			not, _ := fields[i].([]interface{})
			c.Not = &SearchCriteria{}
			if err := c.Not.Parse(not); err != nil {
				return err
			}
		case "OLD":
			c.Old = true
		case "ON":
			i++
			c.On, _ = ParseDate(fields[i])
		case "OR":
			i++
			leftFields, _ := fields[i].([]interface{})

			i++
			rightFields, _ := fields[i].([]interface{})

			c.Or = [2]*SearchCriteria{&SearchCriteria{}, &SearchCriteria{}}
			if err := c.Or[0].Parse(leftFields); err != nil {
				return err
			}
			if err := c.Or[1].Parse(rightFields); err != nil {
				return err
			}
		case "RECENT":
			c.Recent = true
		case "SEEN":
			c.Seen = true
		case "SENTBEFORE":
			i++
			c.SentBefore, _ = ParseDate(fields[i])
		case "SENTON":
			i++
			c.SentOn, _ = ParseDate(fields[i])
		case "SENTSINCE":
			i++
			c.SentSince, _ = ParseDate(fields[i])
		case "SINCE":
			i++
			c.Since, _ = ParseDate(fields[i])
		case "SMALLER":
			i++
			c.Smaller, _ = ParseNumber(fields[i])
		case "SUBJECT":
			i++
			c.Subject, _ = fields[i].(string)
		case "TEXT":
			i++
			c.Text, _ = fields[i].(string)
		case "TO":
			i++
			c.To, _ = fields[i].(string)
		case "UID":
			i++
			s, _ := fields[i].(string)
			c.Uid, _ = NewSeqSet(s)
		case "UNANSWERED":
			c.Unanswered = true
		case "UNDELETED":
			c.Undeleted = true
		case "UNDRAFT":
			c.Undraft = true
		case "UNFLAGGED":
			c.Unflagged = true
		case "UNKEYWORD":
			i++
			c.Unkeyword, _ = fields[i].(string)
		case "UNSEEN":
			c.Unseen = true
		default:
			// Try to parse a sequence set
			var err error
			if c.SeqSet, err = NewSeqSet(f); err != nil {
				return err
			}
		}
	}

	return nil
}

// Format search criteria to fields.
func (c *SearchCriteria) Format() (fields []interface{}) {
	if c.SeqSet != nil {
		fields = append(fields, c.SeqSet)
	}

	if c.Answered {
		fields = append(fields, "ANSWERED")
	}
	if c.Bcc != "" {
		fields = append(fields, "BCC", c.Bcc)
	}
	if !c.Before.IsZero() {
		fields = append(fields, "BEFORE", FormatDate(c.Before))
	}
	if c.Body != "" {
		fields = append(fields, "BODY", c.Body)
	}
	if c.Cc != "" {
		fields = append(fields, "CC", c.Cc)
	}
	if c.Deleted {
		fields = append(fields, "DELETED")
	}
	if c.Draft {
		fields = append(fields, "DRAFT")
	}
	if c.Flagged {
		fields = append(fields, "FLAGGED")
	}
	if c.From != "" {
		fields = append(fields, "FROM", c.From)
	}
	if c.Header[0] != "" && c.Header[1] != "" {
		fields = append(fields, "HEADER", c.Header[0], c.Header[1])
	}
	if c.Keyword != "" {
		fields = append(fields, "KEYWORD", c.Keyword)
	}
	if c.Larger != 0 {
		fields = append(fields, "LARGER", c.Larger)
	}
	if c.New {
		fields = append(fields, "NEW")
	}
	if c.Not != nil {
		fields = append(fields, "NOT", c.Not.Format())
	}
	if c.Old {
		fields = append(fields, "OLD")
	}
	if !c.On.IsZero() {
		fields = append(fields, "ON", FormatDate(c.On))
	}
	if c.Or[0] != nil && c.Or[1] != nil {
		fields = append(fields, "OR", c.Or[0].Format(), c.Or[1].Format())
	}
	if c.Recent {
		fields = append(fields, "RECENT")
	}
	if c.Seen {
		fields = append(fields, "SEEN")
	}
	if !c.SentBefore.IsZero() {
		fields = append(fields, "SENTBEFORE", FormatDate(c.SentBefore))
	}
	if !c.SentOn.IsZero() {
		fields = append(fields, "SENTON", FormatDate(c.SentOn))
	}
	if !c.SentSince.IsZero() {
		fields = append(fields, "SENTSINCE", FormatDate(c.SentSince))
	}
	if !c.Since.IsZero() {
		fields = append(fields, "SINCE", FormatDate(c.Since))
	}
	if c.Smaller != 0 {
		fields = append(fields, "SMALLER", c.Smaller)
	}
	if c.Subject != "" {
		fields = append(fields, "SUBJECT", c.Subject)
	}
	if c.Text != "" {
		fields = append(fields, "TEXT", c.Text)
	}
	if c.To != "" {
		fields = append(fields, "TO", c.To)
	}
	if c.Uid != nil {
		fields = append(fields, "UID", c.Uid)
	}
	if c.Unanswered {
		fields = append(fields, "UNANSWERED")
	}
	if c.Undeleted {
		fields = append(fields, "UNDELETED")
	}
	if c.Undraft {
		fields = append(fields, "UNDRAFT")
	}
	if c.Unflagged {
		fields = append(fields, "UNFLAGGED")
	}
	if c.Unkeyword != "" {
		fields = append(fields, "UNKEYWORD", c.Unkeyword)
	}
	if c.Unseen {
		fields = append(fields, "UNSEEN")
	}

	return
}
