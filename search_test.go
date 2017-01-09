package imap

import (
	"bytes"
	"net/textproto"
	"reflect"
	"testing"
	"time"
)

// Note to myself: writing these boring tests actually fixed 2 bugs :P

var searchSeqSet1, _ = NewSeqSet("1:42")
var searchSeqSet2, _ = NewSeqSet("743:938")
var searchDate1 = time.Date(1997, 11, 21, 0, 0, 0, 0, time.UTC)
var searchDate2 = time.Date(1984, 11, 5, 0, 0, 0, 0, time.UTC)

var searchCriteriaTests = []struct {
	expected string
	criteria *SearchCriteria
}{
	{
		expected: `(1:42 UID 743:938 ` +
			`SINCE "5-Nov-1984" BEFORE "21-Nov-1997" SENTSINCE "5-Nov-1984" SENTBEFORE "21-Nov-1997" ` +
			`FROM root@protonmail.com BODY "hey there" TEXT DILLE ` +
			`ANSWERED DELETED KEYWORD cc UNKEYWORD microsoft ` +
			`LARGER 4242 SMALLER 4342 ` +
			`NOT (SENTON "21-Nov-1997" HEADER Content-Type text/csv) ` +
			`OR (ON "5-Nov-1984" DRAFT FLAGGED UNANSWERED UNDELETED OLD) (UNDRAFT UNFLAGGED UNSEEN))`,
		criteria: &SearchCriteria{
			SeqNum:     searchSeqSet1,
			Uid:        searchSeqSet2,
			Since:      searchDate2,
			Before:     searchDate1,
			SentSince:  searchDate2,
			SentBefore: searchDate1,
			Header: textproto.MIMEHeader{
				"From": {"root@protonmail.com"},
			},
			Body:         []string{"hey there"},
			Text:         []string{"DILLE"},
			WithFlags:    []string{AnsweredFlag, DeletedFlag, "cc"},
			WithoutFlags: []string{"microsoft"},
			Larger:       4242,
			Smaller:      4342,
			Not: []*SearchCriteria{{
				SentSince:  searchDate1,
				SentBefore: searchDate1.Add(24 * time.Hour),
				Header: textproto.MIMEHeader{
					"Content-Type": {"text/csv"},
				},
			}},
			Or: [][2]*SearchCriteria{{
				{
					Since:        searchDate2,
					Before:       searchDate2.Add(24 * time.Hour),
					WithFlags:    []string{DraftFlag, FlaggedFlag},
					WithoutFlags: []string{AnsweredFlag, DeletedFlag, RecentFlag},
				},
				{
					WithoutFlags: []string{DraftFlag, FlaggedFlag, SeenFlag},
				},
			}},
		},
	},
}

func TestSearchCriteria_Format(t *testing.T) {
	for i, test := range searchCriteriaTests {
		fields := test.criteria.Format()

		got, err := formatFields(fields)
		if err != nil {
			t.Fatal("Unexpected no error while formatting fields, got:", err)
		}

		if got != test.expected {
			t.Errorf("Invalid search criteria fields for #%v: got \n%v\n instead of \n%v", i, got, test.expected)
		}
	}
}

func TestSearchCriteria_Parse(t *testing.T) {
	for i, test := range searchCriteriaTests {
		criteria := &SearchCriteria{}

		b := bytes.NewBuffer([]byte(test.expected))
		r := NewReader(b)
		fields, _ := r.ReadFields()

		if err := criteria.Parse(fields[0].([]interface{})); err != nil {
			t.Errorf("Cannot parse search criteria for #%v: %v", i, err)
		} else if !reflect.DeepEqual(criteria, test.criteria) {
			t.Errorf("Invalid search criteria for #%v: got \n%+v\n instead of \n%+v", i, criteria, test.criteria)
		}
	}
}
