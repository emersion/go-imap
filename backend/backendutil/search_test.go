package backendutil

import (
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
)

var matchTests = []struct {
	criteria *imap.SearchCriteria
	res      bool
}{
	{
		criteria: &imap.SearchCriteria{
			Header: textproto.MIMEHeader{"From": {"Mitsuha"}},
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Header: textproto.MIMEHeader{"To": {"Mitsuha"}},
		},
		res: false,
	},
	{
		criteria: &imap.SearchCriteria{SentBefore: testDate.Add(48 * time.Hour)},
		res:      true,
	},
	{
		criteria: &imap.SearchCriteria{
			Not: []*imap.SearchCriteria{{SentSince: testDate.Add(48 * time.Hour)}},
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Not: []*imap.SearchCriteria{{Body: []string{"name"}}},
		},
		res: false,
	},
	{
		criteria: &imap.SearchCriteria{
			Text: []string{"name"},
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{Text: []string{"i'm not in the text"}},
				{Body: []string{"i'm not in the body"}},
			}},
		},
		res: false,
	},
	{
		criteria: &imap.SearchCriteria{
			Header: textproto.MIMEHeader{"Message-Id": {"42@example.org"}},
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Header: textproto.MIMEHeader{"Message-Id": {"43@example.org"}},
		},
		res: false,
	},
	{
		criteria: &imap.SearchCriteria{
			Header: textproto.MIMEHeader{"Message-Id": {""}},
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Header: textproto.MIMEHeader{"Reply-To": {""}},
		},
		res: false,
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
			Header: textproto.MIMEHeader{"Subject": {"your"}},
		},
		res: true,
	},
	{
		criteria: &imap.SearchCriteria{
			Header: textproto.MIMEHeader{"Subject": {"Taki"}},
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

var flagsTests = []struct {
	flags    []string
	criteria *imap.SearchCriteria
	res      bool
}{
	{
		flags: []string{imap.SeenFlag},
		criteria: &imap.SearchCriteria{
			WithFlags:    []string{imap.SeenFlag},
			WithoutFlags: []string{imap.FlaggedFlag},
		},
		res: true,
	},
	{
		flags: []string{imap.SeenFlag},
		criteria: &imap.SearchCriteria{
			WithFlags:    []string{imap.DraftFlag},
			WithoutFlags: []string{imap.FlaggedFlag},
		},
		res: false,
	},
	{
		flags: []string{imap.SeenFlag, imap.FlaggedFlag},
		criteria: &imap.SearchCriteria{
			WithFlags:    []string{imap.SeenFlag},
			WithoutFlags: []string{imap.FlaggedFlag},
		},
		res: false,
	},
	{
		flags: []string{imap.SeenFlag, imap.FlaggedFlag},
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{WithFlags: []string{imap.DraftFlag}},
				{WithoutFlags: []string{imap.SeenFlag}},
			}},
		},
		res: false,
	},
	{
		flags: []string{imap.SeenFlag, imap.FlaggedFlag},
		criteria: &imap.SearchCriteria{
			Not: []*imap.SearchCriteria{
				{WithFlags: []string{imap.SeenFlag}},
			},
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
		Or: [][2]*imap.SearchCriteria{{
			{
				Uid: new(imap.SeqSet),
				Not: []*imap.SearchCriteria{{SeqNum: new(imap.SeqSet)}},
			},
			{
				SeqNum: new(imap.SeqSet),
			},
		}},
	}

	if MatchSeqNumAndUid(seqNum, uid, c) {
		t.Error("Expected not to match criteria")
	}

	c.Or[0][0].Uid.AddNum(uid)
	if !MatchSeqNumAndUid(seqNum, uid, c) {
		t.Error("Expected to match criteria")
	}

	c.Or[0][0].Not[0].SeqNum.AddNum(seqNum)
	if MatchSeqNumAndUid(seqNum, uid, c) {
		t.Error("Expected not to match criteria")
	}

	c.Or[0][1].SeqNum.AddNum(seqNum)
	if !MatchSeqNumAndUid(seqNum, uid, c) {
		t.Error("Expected to match criteria")
	}
}

func TestMatchDate(t *testing.T) {
	date := time.Unix(1483997966, 0)

	c := &imap.SearchCriteria{
		Or: [][2]*imap.SearchCriteria{{
			{
				Since: date.Add(48 * time.Hour),
				Not: []*imap.SearchCriteria{{
					Since: date.Add(48 * time.Hour),
				}},
			},
			{
				Before: date.Add(-48 * time.Hour),
			},
		}},
	}

	if MatchDate(date, c) {
		t.Error("Expected not to match criteria")
	}

	c.Or[0][0].Since = date.Add(-48 * time.Hour)
	if !MatchDate(date, c) {
		t.Error("Expected to match criteria")
	}

	c.Or[0][0].Not[0].Since = date.Add(-48 * time.Hour)
	if MatchDate(date, c) {
		t.Error("Expected not to match criteria")
	}

	c.Or[0][1].Before = date.Add(48 * time.Hour)
	if !MatchDate(date, c) {
		t.Error("Expected to match criteria")
	}
}
