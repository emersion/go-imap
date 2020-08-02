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
			Header: textproto.MIMEHeader{"Totally-Not-Reply-To": {""}},
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

func TestMatchEncoded(t *testing.T) {
	encodedTestMsg := `From: "fox.cpp" <foxcpp@foxcpp.dev>
To: "fox.cpp" <foxcpp@foxcpp.dev>
Subject: =?utf-8?B?0J/RgNC+0LLQtdGA0LrQsCE=?=
Date: Sun, 09 Jun 2019 00:06:43 +0300
MIME-Version: 1.0
Message-ID: <a2aeb99e-52dd-40d3-b99f-1fdaad77ed98@foxcpp.dev>
Content-Type: text/plain; charset=utf-8; format=flowed
Content-Transfer-Encoding: quoted-printable

=D0=AD=D1=82=D0=BE=D1=82 =D1=82=D0=B5=D0=BA=D1=81=D1=82 =D0=B4=D0=BE=D0=BB=
=D0=B6=D0=B5=D0=BD =D0=B1=D1=8B=D1=82=D1=8C =D0=B7=D0=B0=D0=BA=D0=BE=D0=B4=
=D0=B8=D1=80=D0=BE=D0=B2=D0=B0=D0=BD =D0=B2 base64 =D0=B8=D0=BB=D0=B8 quote=
d-encoding.`
	e, err := message.Read(strings.NewReader(encodedTestMsg))
	if err != nil {
		t.Fatal("Expected no error while reading entity, got:", err)
	}

	// Check encoded header.
	crit := imap.SearchCriteria{
		Header: textproto.MIMEHeader{"Subject": []string{"Проверка!"}},
	}

	ok, err := Match(e, 0, 0, time.Now(), []string{}, &crit)
	if err != nil {
		t.Fatal("Expected no error while matching entity, got:", err)
	}

	if !ok {
		t.Error("Expected match for encoded header")
	}

	// Encoded body.
	crit = imap.SearchCriteria{
		Body: []string{"или"},
	}

	ok, err = Match(e, 0, 0, time.Now(), []string{}, &crit)
	if err != nil {
		t.Fatal("Expected no error while matching entity, got:", err)
	}

	if !ok {
		t.Error("Expected match for encoded body")
	}
}

func TestMatchIssue298Regression(t *testing.T) {
	raw1 := "Subject: 1\r\n\r\n1"
	raw2 := "Subject: 2\r\n\r\n22"
	raw3 := "Subject: 3\r\n\r\n333"
	e1, err := message.Read(strings.NewReader(raw1))
	if err != nil {
		t.Fatal("Expected no error while reading entity, got:", err)
	}
	e2, err := message.Read(strings.NewReader(raw2))
	if err != nil {
		t.Fatal("Expected no error while reading entity, got:", err)
	}
	e3, err := message.Read(strings.NewReader(raw3))
	if err != nil {
		t.Fatal("Expected no error while reading entity, got:", err)
	}

	// Search for body size > 15 ("LARGER 15"), which should match messages #2 and #3
	criteria := &imap.SearchCriteria{
		Larger: 15,
	}
	ok1, err := Match(e1, 1, 101, time.Now(), nil, criteria)
	if err != nil {
		t.Fatal("Expected no error while matching entity, got:", err)
	}
	if ok1 {
		t.Errorf("Expected message #1 to not match search criteria")
	}
	ok2, err := Match(e2, 2, 102, time.Now(), nil, criteria)
	if err != nil {
		t.Fatal("Expected no error while matching entity, got:", err)
	}
	if !ok2 {
		t.Errorf("Expected message #2 to match search criteria")
	}
	ok3, err := Match(e3, 3, 103, time.Now(), nil, criteria)
	if err != nil {
		t.Fatal("Expected no error while matching entity, got:", err)
	}
	if !ok3 {
		t.Errorf("Expected message #3 to match search criteria")
	}

	// Search for body size < 17 ("SMALLER 17"), which should match messages #1 and #2
	criteria = &imap.SearchCriteria{
		Smaller: 17,
	}
	ok1, err = Match(e1, 1, 101, time.Now(), nil, criteria)
	if err != nil {
		t.Fatal("Expected no error while matching entity, got:", err)
	}
	if !ok1 {
		t.Errorf("Expected message #1 to match search criteria")
	}
	ok2, err = Match(e2, 2, 102, time.Now(), nil, criteria)
	if err != nil {
		t.Fatal("Expected no error while matching entity, got:", err)
	}
	if !ok2 {
		t.Errorf("Expected message #2 to match search criteria")
	}
	ok3, err = Match(e3, 3, 103, time.Now(), nil, criteria)
	if err != nil {
		t.Fatal("Expected no error while matching entity, got:", err)
	}
	if ok3 {
		t.Errorf("Expected message #3 to not match search criteria")
	}
}
