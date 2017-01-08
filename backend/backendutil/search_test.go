package backendutil

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
)

var matchTests = []struct{
	criteria *imap.SearchCriteria
	res bool
}{
	{
		criteria: &imap.SearchCriteria{From: "Mitsuha"},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{To: "Mitsuha"},
		res: false,
	},
	{
		criteria: &imap.SearchCriteria{Before: testDate.Add(48 * time.Hour)},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Not: &imap.SearchCriteria{Since: testDate.Add(48 * time.Hour)},
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Not: &imap.SearchCriteria{Body: "name"},
		},
		res: false,
	},
	{
		criteria: &imap.SearchCriteria{
			Header: [2]string{"Message-Id", "43@example.org"},
		},
		res: false,
	},
	{
		criteria: &imap.SearchCriteria{
			Header: [2]string{"Message-Id", ""},
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Larger: 10,
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Smaller: 10,
		},
		res: false,
	},
	{
		criteria: &imap.SearchCriteria{
			Subject: "your",
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Subject: "Taki",
		},
		res: false,
	},
}

func TestMatch(t *testing.T) {
	for i, test := range matchTests {
		e, err := message.Read(strings.NewReader(testMailString))
		if err != nil {
			t.Fatal("Expected no error while reading entity, got:", err)
		}

		ok, err := Match(e, test.criteria)
		if err != nil {
			t.Fatal("Expected no error while matching entity, got:", err)
		}

		if test.res && !ok {
			t.Errorf("Expected #%v to match search criteria", i+1)
		}
		if !test.res && ok {
			t.Errorf("Expected #%v not to match search criteria", i+1)
		}
	}
}

var flagsTests = []struct{
	flags []string
	criteria *imap.SearchCriteria
	res bool
}{
	{
		flags: []string{imap.SeenFlag},
		criteria: &imap.SearchCriteria{
			Seen: true,
			Unflagged: true,
		},
		res: true,
	},
	{
		flags: []string{imap.SeenFlag, imap.RecentFlag},
		criteria: &imap.SearchCriteria{New: true},
		res: false,
	},
	{
		flags: []string{imap.RecentFlag},
		criteria: &imap.SearchCriteria{New: true},
		res: true,
	},
	{
		flags: []string{imap.SeenFlag},
		criteria: &imap.SearchCriteria{
			Not: &imap.SearchCriteria{Unseen: true},
		},
		res: true,
	},
	{
		flags: []string{imap.RecentFlag},
		criteria: &imap.SearchCriteria{
			Not: &imap.SearchCriteria{Unseen: true},
		},
		res: false,
	},
}

func TestMatchFlags(t *testing.T) {
	for i, test := range flagsTests {
		ok := MatchFlags(test.flags, test.criteria)
		if test.res && !ok {
			t.Errorf("Expected #%v to match search criteria", i+1)
		}
		if !test.res && ok {
			t.Errorf("Expected #%v not to match search criteria", i+1)
		}
	}
}

func TestMatchSeqNumAndUid(t *testing.T) {
	seqNum := uint32(42)
	uid := uint32(69)

	c := &imap.SearchCriteria{
		Or: [2]*imap.SearchCriteria{
			{
				Uid: new(imap.SeqSet),
				Not: &imap.SearchCriteria{SeqSet: new(imap.SeqSet)},
			},
			{
				SeqSet: new(imap.SeqSet),
			},
		},
	}

	if MatchSeqNumAndUid(seqNum, uid, c) {
		t.Error("Expected not to match criteria")
	}

	c.Or[0].Uid.AddNum(uid)
	if !MatchSeqNumAndUid(seqNum, uid, c) {
		t.Error("Expected to match criteria")
	}

	c.Or[0].Not.SeqSet.AddNum(seqNum)
	if MatchSeqNumAndUid(seqNum, uid, c) {
		t.Error("Expected not to match criteria")
	}

	c.Or[1].SeqSet.AddNum(seqNum)
	if !MatchSeqNumAndUid(seqNum, uid, c) {
		t.Error("Expected to match criteria")
	}
}
