package backendutil

import (
	"testing"

	"github.com/emersion/go-imap"
)

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
			t.Errorf("Expected #%v to match search criteria", i)
		}
		if !test.res && ok {
			t.Errorf("Expected #%v not to match search criteria", i)
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
