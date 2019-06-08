package backendutil

import (
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
)

var testInternalDate = time.Unix(1483997966, 0)

var matchTests = []struct {
	criteria *imap.SearchCriteria
	seqNum   uint32
	uid      uint32
	date     time.Time
	flags    []string
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
	{
		seqNum: 42,
		uid:    69,
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{
					Uid: new(imap.SeqSet),
					Not: []*imap.SearchCriteria{{SeqNum: new(imap.SeqSet)}},
				},
				{
					SeqNum: new(imap.SeqSet),
				},
			}},
		},
		res: false,
	},
	{
		seqNum: 42,
		uid:    69,
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{
					Uid: &imap.SeqSet{Set: []imap.Seq{{69, 69}}},
					Not: []*imap.SearchCriteria{{SeqNum: new(imap.SeqSet)}},
				},
				{
					SeqNum: new(imap.SeqSet),
				},
			}},
		},
		res: true,
	},
	{
		seqNum: 42,
		uid:    69,
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{
					Uid: &imap.SeqSet{Set: []imap.Seq{{69, 69}}},
					Not: []*imap.SearchCriteria{{
						SeqNum: &imap.SeqSet{Set: []imap.Seq{imap.Seq{42, 42}}},
					}},
				},
				{
					SeqNum: new(imap.SeqSet),
				},
			}},
		},
		res: false,
	},
	{
		seqNum: 42,
		uid:    69,
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{
					Uid: &imap.SeqSet{Set: []imap.Seq{{69, 69}}},
					Not: []*imap.SearchCriteria{{
						SeqNum: &imap.SeqSet{Set: []imap.Seq{{42, 42}}},
					}},
				},
				{
					SeqNum: &imap.SeqSet{Set: []imap.Seq{{42, 42}}},
				},
			}},
		},
		res: true,
	},
	{
		date: testInternalDate,
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{
					Since: testInternalDate.Add(48 * time.Hour),
					Not: []*imap.SearchCriteria{{
						Since: testInternalDate.Add(48 * time.Hour),
					}},
				},
				{
					Before: testInternalDate.Add(-48 * time.Hour),
				},
			}},
		},
		res: false,
	},
	{
		date: testInternalDate,
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{
					Since: testInternalDate.Add(-48 * time.Hour),
					Not: []*imap.SearchCriteria{{
						Since: testInternalDate.Add(48 * time.Hour),
					}},
				},
				{
					Before: testInternalDate.Add(-48 * time.Hour),
				},
			}},
		},
		res: true,
	},
	{
		date: testInternalDate,
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{
					Since: testInternalDate.Add(-48 * time.Hour),
					Not: []*imap.SearchCriteria{{
						Since: testInternalDate.Add(-48 * time.Hour),
					}},
				},
				{
					Before: testInternalDate.Add(-48 * time.Hour),
				},
			}},
		},
		res: false,
	},
	{
		date: testInternalDate,
		criteria: &imap.SearchCriteria{
			Or: [][2]*imap.SearchCriteria{{
				{
					Since: testInternalDate.Add(-48 * time.Hour),
					Not: []*imap.SearchCriteria{{
						Since: testInternalDate.Add(-48 * time.Hour),
					}},
				},
				{
					Before: testInternalDate.Add(48 * time.Hour),
				},
			}},
		},
		res: true,
	},
}

func TestMatch(t *testing.T) {
	for i, test := range matchTests {
		e, err := message.Read(strings.NewReader(testMailString))
		if err != nil {
			t.Fatal("Expected no error while reading entity, got:", err)
		}

		ok, err := Match(e, test.seqNum, test.uid, test.date, test.flags, test.criteria)
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
