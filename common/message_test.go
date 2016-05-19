package common_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/emersion/go-imap/common"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		dateStr string
		exp     time.Time
	}{
		{
			"21-Nov-1997 09:55:06 -0600",
			time.Date(1997, 11, 21, 9, 55, 6, 0, time.FixedZone("", -6*60*60)),
		},
	}
	for _, test := range tests {
		date, err := common.ParseDate(test.dateStr)
		if err != nil {
			t.Errorf("Failed parsing %q: %v", test.dateStr, err)
			continue
		}
		if !date.Equal(test.exp) {
			t.Errorf("Parse of %q: got %+v, want %+v", test.dateStr, date, test.exp)
		}
	}
}

var bodyStructureTests = []struct{
	fields []interface{}
	bodyStructure *common.BodyStructure
}{
	{
		fields: []interface{}{"image", "jpeg", nil, "<foo4%25foo1@bar.net>", "A picture of cat", "base64", "4242"},
		bodyStructure: &common.BodyStructure{
			MimeType: "image",
			MimeSubType: "jpeg",
			Params: map[string]string{},
			Id: "<foo4%25foo1@bar.net>",
			Description: "A picture of cat",
			Encoding: "base64",
			Size: 4242,
		},
	},
	{
		fields: []interface{}{"text", "plain", []interface{}{"charset", "utf-8"}, nil, nil, "us-ascii", "42", "2"},
		bodyStructure: &common.BodyStructure{
			MimeType: "text",
			MimeSubType: "plain",
			Params: map[string]string{"charset": "utf-8"},
			Encoding: "us-ascii",
			Size: 42,
			Lines: 2,
		},
	},
	{
		fields: []interface{}{"message", "rfc822", nil, nil, nil, "us-ascii", "42", []interface{}{}, []interface{}{}, "67"},
		bodyStructure: &common.BodyStructure{
			MimeType: "message",
			MimeSubType: "rfc822",
			Params: map[string]string{},
			Encoding: "us-ascii",
			Size: 42,
			Lines: 67,
			Envelope: &common.Envelope{},
			BodyStructure: &common.BodyStructure{},
		},
	},
	{
		fields: []interface{}{"application", "pdf", nil, nil, nil, "base64", "4242", "e0323a9039add2978bf5b49550572c7c", "attachment", "en-US", nil},
		bodyStructure: &common.BodyStructure{
			MimeType: "application",
			MimeSubType: "pdf",
			Params: map[string]string{},
			Encoding: "base64",
			Size: 4242,
			Md5: "e0323a9039add2978bf5b49550572c7c",
			Disposition: "attachment",
			Language: []string{"en-US"},
		},
	},
	{
		fields: []interface{}{
			[]interface{}{"text", "plain", nil, nil, nil, "us-ascii", "87", "22"},
			[]interface{}{"text", "html", nil, nil, nil, "us-ascii", "106", "36"},
			"alternative",
		},
		bodyStructure: &common.BodyStructure{
			MimeType: "multipart",
			MimeSubType: "alternative",
			Params: map[string]string{},
			Parts: []*common.BodyStructure{
				&common.BodyStructure{
					MimeType: "text",
					MimeSubType: "plain",
					Params: map[string]string{},
					Encoding: "us-ascii",
					Size: 87,
					Lines: 22,
				},
				&common.BodyStructure{
					MimeType: "text",
					MimeSubType: "html",
					Params: map[string]string{},
					Encoding: "us-ascii",
					Size: 106,
					Lines: 36,
				},
			},
		},
	},
	{
		fields: []interface{}{
			[]interface{}{"text", "plain", nil, nil, nil, "us-ascii", "87", "22"},
			"alternative", []interface{}{"hello", "world"}, "inline", "en-US", nil,
		},
		bodyStructure: &common.BodyStructure{
			MimeType: "multipart",
			MimeSubType: "alternative",
			Params: map[string]string{"hello": "world"},
			Parts: []*common.BodyStructure{
				&common.BodyStructure{
					MimeType: "text",
					MimeSubType: "plain",
					Params: map[string]string{},
					Encoding: "us-ascii",
					Size: 87,
					Lines: 22,
				},
			},
			Disposition: "inline",
			Language: []string{"en-US"},
		},
	},
}

func TestBodyStructure_Parse(t *testing.T) {
	for i, test := range bodyStructureTests {
		bs := &common.BodyStructure{}

		if err := bs.Parse(test.fields); err != nil {
			t.Errorf("Cannot parse #%v: %v", i, err)
		} else if !reflect.DeepEqual(bs, test.bodyStructure) {
			t.Errorf("Invalid body structure for #%v: %v", i, bs)
		}
	}
}

func TestAddress_Parse(t *testing.T) {
	addr := &common.Address{}
	fields := []interface{}{"The NSA", nil, "root", "nsa.gov"}

	if err := addr.Parse(fields); err != nil {
		t.Fatal(err)
	}

	if addr.PersonalName != "The NSA" {
		t.Error("Invalid personal name:", addr.PersonalName)
	}
	if addr.AtDomainList != "" {
		t.Error("Invalid at-domain-list:", addr.AtDomainList)
	}
	if addr.MailboxName != "root" {
		t.Error("Invalid mailbox name:", addr.MailboxName)
	}
	if addr.HostName != "nsa.gov" {
		t.Error("Invalid host name:", addr.HostName)
	}
}

func TestAddress_Format(t *testing.T) {
	addr := &common.Address{
		PersonalName: "The NSA",
		MailboxName: "root",
		HostName: "nsa.gov",
	}

	fields := addr.Format()
	if len(fields) != 4 {
		t.Fatal("Invalid fields list length")
	}
	if fields[0] != "The NSA" {
		t.Error("Invalid personal name:", fields[0])
	}
	if fields[1] != nil {
		t.Error("Invalid at-domain-list:", fields[1])
	}
	if fields[2] != "root" {
		t.Error("Invalid mailbox name:", fields[2])
	}
	if fields[3] != "nsa.gov" {
		t.Error("Invalid host name:", fields[3])
	}
}
