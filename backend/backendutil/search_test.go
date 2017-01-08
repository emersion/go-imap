package backendutil

import (
	"testing"

	"github.com/emersion/go-imap"
)

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
