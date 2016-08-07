package common

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func formatFields(fields []interface{}) (string, error) {
	b := &bytes.Buffer{}
	w := NewWriter(b).writer()

	if err := w.writeList(fields); err != nil {
		return "", fmt.Errorf("Cannot format %v: %v", fields, err)
	}

	return b.String(), nil
}

func TestParseDate(t *testing.T) {
	tests := []struct{
		dateStr string
		exp     time.Time
	}{
		{
			"21-Nov-1997 09:55:06 -0600",
			time.Date(1997, 11, 21, 9, 55, 6, 0, time.FixedZone("", -6*60*60)),
		},
	}
	for _, test := range tests {
		date, err := ParseDateTime(test.dateStr)
		if err != nil {
			t.Errorf("Failed parsing %q: %v", test.dateStr, err)
			continue
		}
		if !date.Equal(test.exp) {
			t.Errorf("Parse of %q: got %+v, want %+v", test.dateStr, date, test.exp)
		}
	}
}

var messageTests = []struct{
	message *Message
	fields []interface{}
}{
	{
		message: &Message{
			Items: []string{"ENVELOPE", "BODYSTRUCTURE", "FLAGS", "RFC822.SIZE", "UID"},
			Body: map[*BodySectionName]*Literal{},
			Envelope: envelopeTests[0].envelope,
			BodyStructure: bodyStructureTests[0].bodyStructure,
			Flags: []string{SeenFlag, AnsweredFlag},
			Size: 4242,
			Uid: 2424,
		},
		fields: []interface{}{
			"ENVELOPE", envelopeTests[0].fields,
			"BODYSTRUCTURE", bodyStructureTests[0].fields,
			"FLAGS", []interface{}{SeenFlag, AnsweredFlag},
			"RFC822.SIZE", "4242",
			"UID", "2424",
		},
	},
}

func TestMessage_Parse(t *testing.T) {
	for i, test := range messageTests {
		m := NewMessage()
		if err := m.Parse(test.fields); err != nil {
			t.Errorf("Cannot parse message for #%v:", i, err)
		} else if !reflect.DeepEqual(m, test.message) {
			t.Errorf("Invalid parsed message for #%v: got %v but expected %v", i, m, test.message)
		}
	}
}

func TestMessage_Format(t *testing.T) {
	for i, test := range messageTests {
		fields := test.message.Format()

		got, err := formatFields(fields)
		if err != nil {
			t.Error(err)
			continue
		}

		expected, _ := formatFields(test.fields)

		if got != expected {
			t.Errorf("Invalid message fields for #%v: got %v but expected %v", i, got, expected)
		}
	}
}

var bodySectionNameTests = []struct{
	raw string
	parsed *BodySectionName
	formatted string
}{
	{
		raw: "BODY[]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{}},
	},
	{
		raw: "RFC822",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{}},
		formatted: "BODY[]",
	},
	{
		raw: "BODY[HEADER]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Specifier: HeaderSpecifier}},
	},
	{
		raw: "BODY.PEEK[]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{}, Peek: true},
	},
	{
		raw: "BODY[TEXT]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Specifier: TextSpecifier}},
	},
	{
		raw: "RFC822.TEXT",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Specifier: TextSpecifier}},
		formatted: "BODY[TEXT]",
	},
	{
		raw: "RFC822.HEADER",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Specifier: HeaderSpecifier}, Peek: true},
		formatted: "BODY.PEEK[HEADER]",
	},
	{
		raw: "BODY[]<0.512>",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{}, Partial: []int{0, 512}},
	},
	{
		raw: "BODY[1.2.3]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Path: []int{1, 2, 3}}},
	},
	{
		raw: "BODY[1.2.3.HEADER]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Specifier: HeaderSpecifier, Path: []int{1, 2, 3}}},
	},
	{
		raw: "BODY[5.MIME]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Specifier: MimeSpecifier, Path: []int{5}}},
	},
	{
		raw: "BODY[HEADER.FIELDS (From To)]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Specifier: HeaderSpecifier, Fields: []string{"From", "To"}}},
	},
	{
		raw: "BODY[HEADER.FIELDS.NOT (Content-Id)]",
		parsed: &BodySectionName{BodyPartName: &BodyPartName{Specifier: HeaderSpecifier, Fields: []string{"Content-Id"}, NotFields: true}},
	},
}

func TestNewBodySectionName(t *testing.T) {
	for i, test := range bodySectionNameTests {
		bsn, err := NewBodySectionName(test.raw)
		if err != nil {
			t.Errorf("Cannot parse #%v: %v", i, err)
			continue
		}

		if !reflect.DeepEqual(bsn.BodyPartName, test.parsed.BodyPartName) {
			t.Errorf("Invalid body part name for #%v: %#+v", i, bsn.BodyPartName)
		} else if bsn.Peek != test.parsed.Peek {
			t.Errorf("Invalid peek value for #%v: %#+v", i, bsn.Peek)
		} else if !reflect.DeepEqual(bsn.Partial, test.parsed.Partial) {
			t.Errorf("Invalid partial for #%v: %#+v", i, bsn.Partial)
		}
	}
}

func TestBodySectionName_String(t *testing.T) {
	for i, test := range bodySectionNameTests {
		s := test.parsed.String()

		expected := test.formatted
		if expected == "" {
			expected = test.raw
		}

		if expected != s {
			t.Errorf("Invalid body section name for #%v: got %v but expected %v", i, s, expected)
		}
	}
}

func TestBodySectionName_ExtractPartial(t *testing.T) {
	tests := []struct{
		bsn string
		whole string
		partial string
	}{
		{
			bsn: "BODY[]",
			whole: "Hello World!",
			partial: "Hello World!",
		},
		{
			bsn: "BODY[]<6.5>",
			whole: "Hello World!",
			partial: "World",
		},
		{
			bsn: "BODY[]<6.1000>",
			whole: "Hello World!",
			partial: "World!",
		},
		{
			bsn: "BODY[]<0.1>",
			whole: "Hello World!",
			partial: "H",
		},
		{
			bsn: "BODY[]<1000.2000>",
			whole: "Hello World!",
			partial: "",
		},
	}

	for i, test := range tests {
		bsn, err := NewBodySectionName(test.bsn)
		if err != nil {
			t.Errorf("Cannot parse body section name #%v: %v", i, err)
			continue
		}

		partial := string(bsn.ExtractPartial([]byte(test.whole)))
		if partial != test.partial {
			t.Errorf("Invalid partial for #%v: got %v but expected %v", i, partial, test.partial)
		}
	}
}

var t = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.FixedZone("", -6*60*60))

var envelopeTests = []struct{
	envelope *Envelope
	fields []interface{}
}{
	{
		envelope: &Envelope{
			Date: t,
			Subject: "Hello World!",
			From: []*Address{addrTests[0].addr},
			Sender: []*Address{},
			ReplyTo: []*Address{},
			To: []*Address{},
			Cc: []*Address{},
			Bcc: []*Address{},
			InReplyTo: "42@example.org",
			MessageId: "43@example.org",
		},
		fields: []interface{}{
			"10-Nov-2009 23:00:00 -0600",
			"Hello World!",
			[]interface{}{addrTests[0].fields},
			[]interface{}{},
			[]interface{}{},
			[]interface{}{},
			[]interface{}{},
			[]interface{}{},
			"42@example.org",
			"43@example.org",
		},
	},
}

func TestEnvelope_Parse(t *testing.T) {
	for i, test := range envelopeTests {
		e := &Envelope{}

		if err := e.Parse(test.fields); err != nil {
			t.Error("Error parsing envelope:", err)
		} else if !reflect.DeepEqual(e, test.envelope) {
			t.Errorf("Invalid envelope for #%v: got %v but expected %v", i, e, test.envelope)
		}
	}
}

func TestEnvelope_Format(t *testing.T) {
	for i, test := range envelopeTests {
		fields := test.envelope.Format()

		got, err := formatFields(fields)
		if err != nil {
			t.Error(err)
			continue
		}

		expected, _ := formatFields(test.fields)

		if got != expected {
			t.Errorf("Invalid envelope fields for #%v: got %v but expected %v", i, got, expected)
		}
	}
}

var addrTests = []struct{
	fields []interface{}
	addr *Address
}{
	{
		fields: []interface{}{"The NSA", nil, "root", "nsa.gov"},
		addr: &Address{
			PersonalName: "The NSA",
			MailboxName: "root",
			HostName: "nsa.gov",
		},
	},
}

func TestAddress_Parse(t *testing.T) {
	for i, test := range addrTests {
		addr := &Address{}

		if err := addr.Parse(test.fields); err != nil {
			t.Error("Error parsing address:", err)
		} else if !reflect.DeepEqual(addr, test.addr) {
			t.Errorf("Invalid address for #%v: got %v but expected %v", i, addr, test.addr)
		}
	}
}

func TestAddress_Format(t *testing.T) {
	for i, test := range addrTests {
		fields := test.addr.Format()
		if !reflect.DeepEqual(fields, test.fields) {
			t.Errorf("Invalid address fields for #%v: got %v but expected %v", i, fields, test.fields)
		}
	}
}

func TestAddressList(t *testing.T) {
	fields := make([]interface{}, len(addrTests))
	addrs := make([]*Address, len(addrTests))
	for i, test := range addrTests {
		fields[i] = test.fields
		addrs[i] = test.addr
	}

	gotAddrs := ParseAddressList(fields)
	if !reflect.DeepEqual(gotAddrs, addrs) {
		t.Error("Invalid address list: got", gotAddrs, "but expected", addrs)
	}

	gotFields := FormatAddressList(addrs)
	if !reflect.DeepEqual(gotFields, fields) {
		t.Error("Invalid address list fields: got", gotFields, "but expected", fields)
	}
}

var paramsListTest = []struct{
	fields []interface{}
	params map[string]string
}{
	{
		fields: []interface{}{},
		params: map[string]string{},
	},
	{
		fields: []interface{}{"a", "b"},
		params: map[string]string{"a": "b"},
	},
}

func TestParseParamList(t *testing.T) {
	for i, test := range paramsListTest {
		if params, err := ParseParamList(test.fields); err != nil {
			t.Errorf("Cannot parse params fields for #%v: %v", i, err)
		} else if !reflect.DeepEqual(params, test.params) {
			t.Errorf("Invalid params for #%v: got %v but expected %v", i, params, test.params)
		}
	}

	// Malformed params lists

	fields := []interface{}{"cc", []interface{}{"dille"}}
	if params, err := ParseParamList(fields); err == nil {
		t.Error("Parsed invalid params list:", params)
	}

	fields = []interface{}{"cc"}
	if params, err := ParseParamList(fields); err == nil {
		t.Error("Parsed invalid params list:", params)
	}
}

func TestFormatParamList(t *testing.T) {
	for i, test := range paramsListTest {
		fields := FormatParamList(test.params)

		if !reflect.DeepEqual(fields, test.fields) {
			t.Errorf("Invalid params fields for #%v: got %v but expected %v", i, fields, test.fields)
		}
	}
}

var bodyStructureTests = []struct{
	fields []interface{}
	bodyStructure *BodyStructure
}{
	{
		fields: []interface{}{"image", "jpeg", []interface{}{}, "<foo4%25foo1@bar.net>", "A picture of cat", "base64", "4242"},
		bodyStructure: &BodyStructure{
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
		bodyStructure: &BodyStructure{
			MimeType: "text",
			MimeSubType: "plain",
			Params: map[string]string{"charset": "utf-8"},
			Encoding: "us-ascii",
			Size: 42,
			Lines: 2,
		},
	},
	{
		fields: []interface{}{
			"message", "rfc822", []interface{}{}, nil, nil, "us-ascii", "42",
			(&Envelope{}).Format(),
			(&BodyStructure{}).Format(),
			"67",
		},
		bodyStructure: &BodyStructure{
			MimeType: "message",
			MimeSubType: "rfc822",
			Params: map[string]string{},
			Encoding: "us-ascii",
			Size: 42,
			Lines: 67,
			Envelope: &Envelope{
				From: []*Address{},
				Sender: []*Address{},
				ReplyTo: []*Address{},
				To: []*Address{},
				Cc: []*Address{},
				Bcc: []*Address{},
			},
			BodyStructure: &BodyStructure{
				Params: map[string]string{},
			},
		},
	},
	{
		fields: []interface{}{"application", "pdf", []interface{}{}, nil, nil, "base64", "4242",
			"e0323a9039add2978bf5b49550572c7c", "attachment", []interface{}{"en-US"}, []interface{}{}},
		bodyStructure: &BodyStructure{
			MimeType: "application",
			MimeSubType: "pdf",
			Params: map[string]string{},
			Encoding: "base64",
			Size: 4242,
			Extended: true,
			Md5: "e0323a9039add2978bf5b49550572c7c",
			Disposition: "attachment",
			Language: []string{"en-US"},
			Location: []string{},
		},
	},
	{
		fields: []interface{}{
			[]interface{}{"text", "plain", []interface{}{}, nil, nil, "us-ascii", "87", "22"},
			[]interface{}{"text", "html", []interface{}{}, nil, nil, "us-ascii", "106", "36"},
			"alternative",
		},
		bodyStructure: &BodyStructure{
			MimeType: "multipart",
			MimeSubType: "alternative",
			Params: map[string]string{},
			Parts: []*BodyStructure{
				&BodyStructure{
					MimeType: "text",
					MimeSubType: "plain",
					Params: map[string]string{},
					Encoding: "us-ascii",
					Size: 87,
					Lines: 22,
				},
				&BodyStructure{
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
			[]interface{}{"text", "plain", []interface{}{}, nil, nil, "us-ascii", "87", "22"},
			"alternative", []interface{}{"hello", "world"}, "inline", []interface{}{"en-US"}, []interface{}{},
		},
		bodyStructure: &BodyStructure{
			MimeType: "multipart",
			MimeSubType: "alternative",
			Params: map[string]string{"hello": "world"},
			Parts: []*BodyStructure{
				&BodyStructure{
					MimeType: "text",
					MimeSubType: "plain",
					Params: map[string]string{},
					Encoding: "us-ascii",
					Size: 87,
					Lines: 22,
				},
			},
			Extended: true,
			Disposition: "inline",
			Language: []string{"en-US"},
			Location: []string{},
		},
	},
}

func TestBodyStructure_Parse(t *testing.T) {
	for i, test := range bodyStructureTests {
		bs := &BodyStructure{}

		if err := bs.Parse(test.fields); err != nil {
			t.Errorf("Cannot parse #%v: %v", i, err)
		} else if !reflect.DeepEqual(bs, test.bodyStructure) {
			t.Errorf("Invalid body structure for #%v: got %v but expected %v", i, bs, test.bodyStructure)
		}
	}
}

func TestBodyStructure_Format(t *testing.T) {
	for i, test := range bodyStructureTests {
		fields := test.bodyStructure.Format()
		got, err := formatFields(fields)
		if err != nil {
			t.Error(err)
			continue
		}

		expected, _ := formatFields(test.fields)

		if got != expected {
			t.Errorf("Invalid body structure fields for #%v: has %v but expected %v", i, got, expected)
		}
	}
}
