package imap

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestCanonicalFlag(t *testing.T) {
	if got := CanonicalFlag("\\SEEN"); got != SeenFlag {
		t.Errorf("Invalid canonical flag: expected %q but got %q", SeenFlag, got)
	}

	if got := CanonicalFlag("Junk"); got != "junk" {
		t.Errorf("Invalid canonical flag: expected %q but got %q", "junk", got)
	}
}

func TestNewMessage(t *testing.T) {
	msg := NewMessage(42, []FetchItem{FetchBodyStructure, FetchFlags})

	expected := &Message{
		SeqNum:     42,
		Items:      map[FetchItem]interface{}{FetchBodyStructure: nil, FetchFlags: nil},
		Body:       make(map[*BodySectionName]Literal),
		itemsOrder: []FetchItem{FetchBodyStructure, FetchFlags},
	}

	if !reflect.DeepEqual(expected, msg) {
		t.Errorf("Invalid message: expected \n%+v\n but got \n%+v", expected, msg)
	}
}

func formatFields(fields []interface{}) (string, error) {
	b := &bytes.Buffer{}
	w := NewWriter(b)

	if err := w.writeList(fields); err != nil {
		return "", fmt.Errorf("Cannot format \n%+v\n got error: \n%v", fields, err)
	}

	return b.String(), nil
}

var messageTests = []struct {
	message *Message
	fields  []interface{}
}{
	{
		message: &Message{
			Items: map[FetchItem]interface{}{
				FetchEnvelope:   nil,
				FetchBody:       nil,
				FetchFlags:      nil,
				FetchRFC822Size: nil,
				FetchUid:        nil,
			},
			Body:          map[*BodySectionName]Literal{},
			Envelope:      envelopeTests[0].envelope,
			BodyStructure: bodyStructureTests[0].bodyStructure,
			Flags:         []string{SeenFlag, AnsweredFlag},
			Size:          4242,
			Uid:           2424,
			itemsOrder:    []FetchItem{FetchEnvelope, FetchBody, FetchFlags, FetchRFC822Size, FetchUid},
		},
		fields: []interface{}{
			RawString("ENVELOPE"), envelopeTests[0].fields,
			RawString("BODY"), bodyStructureTests[0].fields,
			RawString("FLAGS"), []interface{}{RawString(SeenFlag), RawString(AnsweredFlag)},
			RawString("RFC822.SIZE"), RawString("4242"),
			RawString("UID"), RawString("2424"),
		},
	},
}

func TestMessage_Parse(t *testing.T) {
	for i, test := range messageTests {
		m := &Message{}
		if err := m.Parse(test.fields); err != nil {
			t.Errorf("Cannot parse message for #%v: %v", i, err)
		} else if !reflect.DeepEqual(m, test.message) {
			t.Errorf("Invalid parsed message for #%v: got \n%+v\n but expected \n%+v", i, m, test.message)
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
			t.Errorf("Invalid message fields for #%v: got \n%v\n but expected \n%v", i, got, expected)
		}
	}
}

var bodySectionNameTests = []struct {
	raw       string
	parsed    *BodySectionName
	formatted string
}{
	{
		raw:    "BODY[]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{}},
	},
	{
		raw:       "RFC822",
		parsed:    &BodySectionName{BodyPartName: BodyPartName{}},
		formatted: "BODY[]",
	},
	{
		raw:    "BODY[HEADER]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{Specifier: HeaderSpecifier}},
	},
	{
		raw:    "BODY.PEEK[]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{}, Peek: true},
	},
	{
		raw:    "BODY[TEXT]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{Specifier: TextSpecifier}},
	},
	{
		raw:       "RFC822.TEXT",
		parsed:    &BodySectionName{BodyPartName: BodyPartName{Specifier: TextSpecifier}},
		formatted: "BODY[TEXT]",
	},
	{
		raw:       "RFC822.HEADER",
		parsed:    &BodySectionName{BodyPartName: BodyPartName{Specifier: HeaderSpecifier}, Peek: true},
		formatted: "BODY.PEEK[HEADER]",
	},
	{
		raw:    "BODY[]<0.512>",
		parsed: &BodySectionName{BodyPartName: BodyPartName{}, Partial: []int{0, 512}},
	},
	{
		raw:    "BODY[]<512>",
		parsed: &BodySectionName{BodyPartName: BodyPartName{}, Partial: []int{512}},
	},
	{
		raw:    "BODY[1.2.3]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{Path: []int{1, 2, 3}}},
	},
	{
		raw:    "BODY[1.2.3.HEADER]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{Specifier: HeaderSpecifier, Path: []int{1, 2, 3}}},
	},
	{
		raw:    "BODY[5.MIME]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{Specifier: MIMESpecifier, Path: []int{5}}},
	},
	{
		raw:    "BODY[HEADER.FIELDS (From To)]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{Specifier: HeaderSpecifier, Fields: []string{"From", "To"}}},
	},
	{
		raw:    "BODY[HEADER.FIELDS.NOT (Content-Id)]",
		parsed: &BodySectionName{BodyPartName: BodyPartName{Specifier: HeaderSpecifier, Fields: []string{"Content-Id"}, NotFields: true}},
	},
}

func TestNewBodySectionName(t *testing.T) {
	for i, test := range bodySectionNameTests {
		bsn, err := ParseBodySectionName(FetchItem(test.raw))
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
		s := string(test.parsed.FetchItem())

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
	tests := []struct {
		bsn     string
		whole   string
		partial string
	}{
		{
			bsn:     "BODY[]",
			whole:   "Hello World!",
			partial: "Hello World!",
		},
		{
			bsn:     "BODY[]<6.5>",
			whole:   "Hello World!",
			partial: "World",
		},
		{
			bsn:     "BODY[]<6.1000>",
			whole:   "Hello World!",
			partial: "World!",
		},
		{
			bsn:     "BODY[]<0.1>",
			whole:   "Hello World!",
			partial: "H",
		},
		{
			bsn:     "BODY[]<1000.2000>",
			whole:   "Hello World!",
			partial: "",
		},
	}

	for i, test := range tests {
		bsn, err := ParseBodySectionName(FetchItem(test.bsn))
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

var envelopeTests = []struct {
	envelope *Envelope
	fields   []interface{}
}{
	{
		envelope: &Envelope{
			Date:      t,
			Subject:   "Hello World!",
			From:      []*Address{addrTests[0].addr},
			Sender:    nil,
			ReplyTo:   nil,
			To:        nil,
			Cc:        nil,
			Bcc:       nil,
			InReplyTo: "42@example.org",
			MessageId: "43@example.org",
		},
		fields: []interface{}{
			"Tue, 10 Nov 2009 23:00:00 -0600",
			"Hello World!",
			[]interface{}{addrTests[0].fields},
			nil,
			nil,
			nil,
			nil,
			nil,
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

func TestEnvelope_Parse_literal(t *testing.T) {
	subject := "Hello World!"
	l := bytes.NewBufferString(subject)
	fields := []interface{}{
		"Tue, 10 Nov 2009 23:00:00 -0600",
		l,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		"42@example.org",
		"43@example.org",
	}

	e := &Envelope{}
	if err := e.Parse(fields); err != nil {
		t.Error("Error parsing envelope:", err)
	} else if e.Subject != subject {
		t.Errorf("Invalid envelope subject: got %v but expected %v", e.Subject, subject)
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

var addrTests = []struct {
	fields []interface{}
	addr   *Address
}{
	{
		fields: []interface{}{"The NSA", nil, "root", "nsa.gov"},
		addr: &Address{
			PersonalName: "The NSA",
			MailboxName:  "root",
			HostName:     "nsa.gov",
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

func TestEmptyAddress(t *testing.T) {
	fields := []interface{}{nil, nil, nil, nil}
	addr := &Address{}
	err := addr.Parse(fields)
	if err == nil {
		t.Error("A nil address did not return an error")
	}
}

func TestEmptyGroupAddress(t *testing.T) {
	fields := []interface{}{nil, nil, "undisclosed-recipients", nil}
	addr := &Address{}
	err := addr.Parse(fields)
	if err == nil {
		t.Error("An empty group did not return an error when parsed as address")
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

func TestEmptyAddressList(t *testing.T) {
	addrs := make([]*Address, 0)

	gotFields := FormatAddressList(addrs)
	if !reflect.DeepEqual(gotFields, nil) {
		t.Error("Invalid address list fields: got", gotFields, "but expected nil")
	}
}

var paramsListTest = []struct {
	fields []interface{}
	params map[string]string
}{
	{
		fields: nil,
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

var bodyStructureTests = []struct {
	fields        []interface{}
	bodyStructure *BodyStructure
}{
	{
		fields: []interface{}{"image", "jpeg", []interface{}{}, "<foo4%25foo1@bar.net>", "A picture of cat", "base64", RawString("4242")},
		bodyStructure: &BodyStructure{
			MIMEType:    "image",
			MIMESubType: "jpeg",
			Params:      map[string]string{},
			Id:          "<foo4%25foo1@bar.net>",
			Description: "A picture of cat",
			Encoding:    "base64",
			Size:        4242,
		},
	},
	{
		fields: []interface{}{"text", "plain", []interface{}{"charset", "utf-8"}, nil, nil, "us-ascii", RawString("42"), RawString("2")},
		bodyStructure: &BodyStructure{
			MIMEType:    "text",
			MIMESubType: "plain",
			Params:      map[string]string{"charset": "utf-8"},
			Encoding:    "us-ascii",
			Size:        42,
			Lines:       2,
		},
	},
	{
		fields: []interface{}{
			"message", "rfc822", []interface{}{}, nil, nil, "us-ascii", RawString("42"),
			(&Envelope{}).Format(),
			(&BodyStructure{}).Format(),
			RawString("67"),
		},
		bodyStructure: &BodyStructure{
			MIMEType:    "message",
			MIMESubType: "rfc822",
			Params:      map[string]string{},
			Encoding:    "us-ascii",
			Size:        42,
			Lines:       67,
			Envelope: &Envelope{
				From:    nil,
				Sender:  nil,
				ReplyTo: nil,
				To:      nil,
				Cc:      nil,
				Bcc:     nil,
			},
			BodyStructure: &BodyStructure{
				Params: map[string]string{},
			},
		},
	},
	{
		fields: []interface{}{
			"application", "pdf", []interface{}{}, nil, nil, "base64", RawString("4242"),
			"e0323a9039add2978bf5b49550572c7c",
			[]interface{}{"attachment", []interface{}{"filename", "document.pdf"}},
			[]interface{}{"en-US"}, []interface{}{},
		},
		bodyStructure: &BodyStructure{
			MIMEType:          "application",
			MIMESubType:       "pdf",
			Params:            map[string]string{},
			Encoding:          "base64",
			Size:              4242,
			Extended:          true,
			MD5:               "e0323a9039add2978bf5b49550572c7c",
			Disposition:       "attachment",
			DispositionParams: map[string]string{"filename": "document.pdf"},
			Language:          []string{"en-US"},
			Location:          []string{},
		},
	},
	{
		fields: []interface{}{
			[]interface{}{"text", "plain", []interface{}{}, nil, nil, "us-ascii", RawString("87"), RawString("22")},
			[]interface{}{"text", "html", []interface{}{}, nil, nil, "us-ascii", RawString("106"), RawString("36")},
			"alternative",
		},
		bodyStructure: &BodyStructure{
			MIMEType:    "multipart",
			MIMESubType: "alternative",
			Params:      map[string]string{},
			Parts: []*BodyStructure{
				{
					MIMEType:    "text",
					MIMESubType: "plain",
					Params:      map[string]string{},
					Encoding:    "us-ascii",
					Size:        87,
					Lines:       22,
				},
				{
					MIMEType:    "text",
					MIMESubType: "html",
					Params:      map[string]string{},
					Encoding:    "us-ascii",
					Size:        106,
					Lines:       36,
				},
			},
		},
	},
	{
		fields: []interface{}{
			[]interface{}{"text", "plain", []interface{}{}, nil, nil, "us-ascii", RawString("87"), RawString("22")},
			"alternative", []interface{}{"hello", "world"},
			[]interface{}{"inline", []interface{}{}},
			[]interface{}{"en-US"}, []interface{}{},
		},
		bodyStructure: &BodyStructure{
			MIMEType:    "multipart",
			MIMESubType: "alternative",
			Params:      map[string]string{"hello": "world"},
			Parts: []*BodyStructure{
				{
					MIMEType:    "text",
					MIMESubType: "plain",
					Params:      map[string]string{},
					Encoding:    "us-ascii",
					Size:        87,
					Lines:       22,
				},
			},
			Extended:          true,
			Disposition:       "inline",
			DispositionParams: map[string]string{},
			Language:          []string{"en-US"},
			Location:          []string{},
		},
	},
}

func TestBodyStructure_Parse(t *testing.T) {
	for i, test := range bodyStructureTests {
		bs := &BodyStructure{}

		if err := bs.Parse(test.fields); err != nil {
			t.Errorf("Cannot parse #%v: %v", i, err)
		} else if !reflect.DeepEqual(bs, test.bodyStructure) {
			t.Errorf("Invalid body structure for #%v: got \n%+v\n but expected \n%+v", i, bs, test.bodyStructure)
		}
	}
}

func TestBodyStructure_Parse_uppercase(t *testing.T) {
	fields := []interface{}{
		"APPLICATION", "PDF", []interface{}{"NAME", "Document.pdf"}, nil, nil,
		"BASE64", RawString("4242"), nil,
		[]interface{}{"ATTACHMENT", []interface{}{"FILENAME", "Document.pdf"}},
		nil, nil,
	}

	expected := &BodyStructure{
		MIMEType:          "application",
		MIMESubType:       "pdf",
		Params:            map[string]string{"name": "Document.pdf"},
		Encoding:          "base64",
		Size:              4242,
		Extended:          true,
		MD5:               "",
		Disposition:       "attachment",
		DispositionParams: map[string]string{"filename": "Document.pdf"},
		Language:          nil,
		Location:          []string{},
	}

	bs := &BodyStructure{}
	if err := bs.Parse(fields); err != nil {
		t.Errorf("Cannot parse: %v", err)
	} else if !reflect.DeepEqual(bs, expected) {
		t.Errorf("Invalid body structure: got \n%+v\n but expected \n%+v", bs, expected)
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
			t.Errorf("Invalid body structure fields for #%v: has \n%v\n but expected \n%v", i, got, expected)
		}
	}
}

func TestBodyStructureFilename(t *testing.T) {
	tests := []struct {
		bs       BodyStructure
		filename string
	}{
		{
			bs: BodyStructure{
				DispositionParams: map[string]string{"filename": "cat.png"},
			},
			filename: "cat.png",
		},
		{
			bs: BodyStructure{
				Params: map[string]string{"name": "cat.png"},
			},
			filename: "cat.png",
		},
		{
			bs:       BodyStructure{},
			filename: "",
		},
		{
			bs: BodyStructure{
				DispositionParams: map[string]string{"filename": "=?UTF-8?Q?Opis_przedmiotu_zam=c3=b3wienia_-_za=c5=82=c4=85cznik_nr_1?= =?UTF-8?Q?=2epdf?="},
			},
			filename: "Opis przedmiotu zamówienia - załącznik nr 1.pdf",
		},
	}

	for i, test := range tests {
		got, err := test.bs.Filename()
		if err != nil {
			t.Errorf("Invalid body structure filename for #%v: error: %v", i, err)
			continue
		}

		if got != test.filename {
			t.Errorf("Invalid body structure filename for #%v: got '%v', want '%v'", i, got, test.filename)
		}
	}
}

func TestBodyStructureWalk(t *testing.T) {
	textPlain := &BodyStructure{
		MIMEType:    "text",
		MIMESubType: "plain",
	}

	textHTML := &BodyStructure{
		MIMEType:    "text",
		MIMESubType: "plain",
	}

	multipartAlternative := &BodyStructure{
		MIMEType:    "multipart",
		MIMESubType: "alternative",
		Parts:       []*BodyStructure{textPlain, textHTML},
	}

	imagePNG := &BodyStructure{
		MIMEType:    "image",
		MIMESubType: "png",
	}

	multipartMixed := &BodyStructure{
		MIMEType:    "multipart",
		MIMESubType: "mixed",
		Parts:       []*BodyStructure{multipartAlternative, imagePNG},
	}

	type testNode struct {
		path []int
		part *BodyStructure
	}

	tests := []struct {
		bs           *BodyStructure
		nodes        []testNode
		walkChildren bool
	}{
		{
			bs: textPlain,
			nodes: []testNode{
				{path: []int{1}, part: textPlain},
			},
		},
		{
			bs: multipartAlternative,
			nodes: []testNode{
				{path: nil, part: multipartAlternative},
				{path: []int{1}, part: textPlain},
				{path: []int{2}, part: textHTML},
			},
			walkChildren: true,
		},
		{
			bs: multipartMixed,
			nodes: []testNode{
				{path: nil, part: multipartMixed},
				{path: []int{1}, part: multipartAlternative},
				{path: []int{1, 1}, part: textPlain},
				{path: []int{1, 2}, part: textHTML},
				{path: []int{2}, part: imagePNG},
			},
			walkChildren: true,
		},
		{
			bs: multipartMixed,
			nodes: []testNode{
				{path: nil, part: multipartMixed},
			},
			walkChildren: false,
		},
	}

	for i, test := range tests {
		j := 0
		test.bs.Walk(func(path []int, part *BodyStructure) bool {
			if j >= len(test.nodes) {
				t.Errorf("Test #%v: invalid node count: got > %v, want %v", i, j, len(test.nodes))
				return false
			}
			n := &test.nodes[j]
			if !reflect.DeepEqual(path, n.path) {
				t.Errorf("Test #%v: node #%v: invalid path: got %v, want %v", i, j, path, n.path)
			}
			if part != n.part {
				t.Errorf("Test #%v: node #%v: invalid part: got %v, want %v", i, j, part, n.part)
			}
			j++
			return test.walkChildren
		})
		if j != len(test.nodes) {
			t.Errorf("Test #%v: invalid node count: got %v, want %v", i, j, len(test.nodes))
		}
	}
}
