package imap

import (
	"reflect"
	"testing"
	"time"
)

// Note to myself: writing these broing tests actually fixed 2 bugs :P

var searchSeqSet1, _ = NewSeqSet("1:42")
var searchSeqSet2, _ = NewSeqSet("743:938")
var searchDate1 = time.Date(1997, 11, 21, 0, 0, 0, 0, time.UTC)
var searchDate2 = time.Date(1984, 11, 5, 0, 0, 0, 0, time.UTC)

var searchCriteriaTests = []struct {
	fields   []interface{}
	criteria *SearchCriteria
}{
	{
		fields: []interface{}{
			"1:42",
			"ANSWERED",
			"BCC", "root@nsa.gov",
			"BEFORE", "21-Nov-1997",
			"BODY", "hey there",
			"CC", "root@gchq.gov.uk",
			"DELETED",
			"DRAFT",
			"FLAGGED",
			"FROM", "root@protonmail.com",
			"HEADER", "Content-Type", "text/csv",
			"KEYWORD", "cc",
			"LARGER", "4242",
			"NEW",
			"NOT", []interface{}{"OLD", "ON", " 5-Nov-1984"},
			"OR", []interface{}{"RECENT", "SENTON", "21-Nov-1997"}, []interface{}{"SEEN", "SENTBEFORE", " 5-Nov-1984"},
			"SENTSINCE", "21-Nov-1997",
			"SINCE", " 5-Nov-1984",
			"SMALLER", "643",
			"SUBJECT", "saucisse royale",
			"TEXT", "DILLE",
			"TO", "cc@dille.cc",
			"UID", "743:938",
			"UNANSWERED",
			"UNDELETED",
			"UNDRAFT",
			"UNFLAGGED",
			"UNKEYWORD", "microsoft",
			"UNSEEN",
		},
		criteria: &SearchCriteria{
			SeqSet:   searchSeqSet1,
			Answered: true,
			Bcc:      "root@nsa.gov",
			Before:   searchDate1,
			Body:     "hey there",
			Cc:       "root@gchq.gov.uk",
			Deleted:  true,
			Draft:    true,
			Flagged:  true,
			From:     "root@protonmail.com",
			Header:   [2]string{"Content-Type", "text/csv"},
			Keyword:  "cc",
			Larger:   4242,
			New:      true,
			Not:      &SearchCriteria{Old: true, On: searchDate2},
			Or: [2]*SearchCriteria{
				&SearchCriteria{Recent: true, SentOn: searchDate1},
				&SearchCriteria{Seen: true, SentBefore: searchDate2},
			},
			SentSince:  searchDate1,
			Since:      searchDate2,
			Smaller:    643,
			Subject:    "saucisse royale",
			Text:       "DILLE",
			To:         "cc@dille.cc",
			Uid:        searchSeqSet2,
			Unanswered: true,
			Undeleted:  true,
			Undraft:    true,
			Unflagged:  true,
			Unkeyword:  "microsoft",
			Unseen:     true,
		},
	},
}

func TestSearchCriteria_Format(t *testing.T) {
	for i, test := range searchCriteriaTests {
		fields := test.criteria.Format()

		got, _ := formatFields(fields)
		expected, _ := formatFields(test.fields)

		if got != expected {
			t.Errorf("Invalid search criteria fields for #%v: got %v instead of %v", i, got, expected)
		}
	}
}

func TestSearchCriteria_Parse(t *testing.T) {
	for i, test := range searchCriteriaTests {
		criteria := &SearchCriteria{}

		if err := criteria.Parse(test.fields); err != nil {
			t.Errorf("Cannot parse search criteria for #%v: %v", i, err)
		} else if !reflect.DeepEqual(criteria, test.criteria) {
			t.Errorf("Invalid search criteria for #%v: got %v instead of %v", i, criteria, test.criteria)
		}
	}
}
