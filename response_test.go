package imap_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/emersion/go-imap"
)

func TestResp_WriteTo(t *testing.T) {
	var b bytes.Buffer
	w := imap.NewWriter(&b)

	resp := imap.NewUntaggedResp([]interface{}{imap.RawString("76"), imap.RawString("FETCH"), []interface{}{imap.RawString("UID"), 783}})
	if err := resp.WriteTo(w); err != nil {
		t.Fatal(err)
	}

	if b.String() != "* 76 FETCH (UID 783)\r\n" {
		t.Error("Invalid response:", b.String())
	}
}

func TestContinuationReq_WriteTo(t *testing.T) {
	var b bytes.Buffer
	w := imap.NewWriter(&b)

	resp := &imap.ContinuationReq{}

	if err := resp.WriteTo(w); err != nil {
		t.Fatal(err)
	}

	if b.String() != "+\r\n" {
		t.Error("Invalid continuation:", b.String())
	}
}

func TestContinuationReq_WriteTo_WithInfo(t *testing.T) {
	var b bytes.Buffer
	w := imap.NewWriter(&b)

	resp := &imap.ContinuationReq{Info: "send literal"}

	if err := resp.WriteTo(w); err != nil {
		t.Fatal(err)
	}

	if b.String() != "+ send literal\r\n" {
		t.Error("Invalid continuation:", b.String())
	}
}

func TestReadResp_ContinuationReq(t *testing.T) {
	b := bytes.NewBufferString("+ send literal\r\n")
	r := imap.NewReader(b)

	resp, err := imap.ReadResp(r)
	if err != nil {
		t.Fatal(err)
	}

	cont, ok := resp.(*imap.ContinuationReq)
	if !ok {
		t.Fatal("Response is not a continuation request")
	}

	if cont.Info != "send literal" {
		t.Error("Invalid info:", cont.Info)
	}
}

func TestReadResp_ContinuationReq_NoInfo(t *testing.T) {
	b := bytes.NewBufferString("+\r\n")
	r := imap.NewReader(b)

	resp, err := imap.ReadResp(r)
	if err != nil {
		t.Fatal(err)
	}

	cont, ok := resp.(*imap.ContinuationReq)
	if !ok {
		t.Fatal("Response is not a continuation request")
	}

	if cont.Info != "" {
		t.Error("Invalid info:", cont.Info)
	}
}

func TestReadResp_Resp(t *testing.T) {
	b := bytes.NewBufferString("* 1 EXISTS\r\n")
	r := imap.NewReader(b)

	resp, err := imap.ReadResp(r)
	if err != nil {
		t.Fatal(err)
	}

	data, ok := resp.(*imap.DataResp)
	if !ok {
		t.Fatal("Invalid response type")
	}

	if data.Tag != "*" {
		t.Error("Invalid tag:", data.Tag)
	}
	if len(data.Fields) != 2 {
		t.Error("Invalid fields:", data.Fields)
	}
}

func TestReadResp_Resp_NoArgs(t *testing.T) {
	b := bytes.NewBufferString("* SEARCH\r\n")
	r := imap.NewReader(b)

	resp, err := imap.ReadResp(r)
	if err != nil {
		t.Fatal(err)
	}

	data, ok := resp.(*imap.DataResp)
	if !ok {
		t.Fatal("Invalid response type")
	}

	if data.Tag != "*" {
		t.Error("Invalid tag:", data.Tag)
	}
	if len(data.Fields) != 1 || data.Fields[0] != "SEARCH" {
		t.Error("Invalid fields:", data.Fields)
	}
}

func TestReadResp_StatusResp(t *testing.T) {
	tests := []struct {
		input    string
		expected *imap.StatusResp
	}{
		{
			input: "* OK IMAP4rev1 Service Ready\r\n",
			expected: &imap.StatusResp{
				Tag:  "*",
				Type: imap.StatusRespOk,
				Info: "IMAP4rev1 Service Ready",
			},
		},
		{
			input: "* PREAUTH Welcome Pauline!\r\n",
			expected: &imap.StatusResp{
				Tag:  "*",
				Type: imap.StatusRespPreauth,
				Info: "Welcome Pauline!",
			},
		},
		{
			input: "a001 OK NOOP completed\r\n",
			expected: &imap.StatusResp{
				Tag:  "a001",
				Type: imap.StatusRespOk,
				Info: "NOOP completed",
			},
		},
		{
			input: "a001 OK [READ-ONLY] SELECT completed\r\n",
			expected: &imap.StatusResp{
				Tag:  "a001",
				Type: imap.StatusRespOk,
				Code: "READ-ONLY",
				Info: "SELECT completed",
			},
		},
		{
			input: "a001 OK [CAPABILITY IMAP4rev1 UIDPLUS] LOGIN completed\r\n",
			expected: &imap.StatusResp{
				Tag:       "a001",
				Type:      imap.StatusRespOk,
				Code:      "CAPABILITY",
				Arguments: []interface{}{"IMAP4rev1", "UIDPLUS"},
				Info:      "LOGIN completed",
			},
		},
	}

	for _, test := range tests {
		b := bytes.NewBufferString(test.input)
		r := imap.NewReader(b)

		resp, err := imap.ReadResp(r)
		if err != nil {
			t.Fatal(err)
		}

		status, ok := resp.(*imap.StatusResp)
		if !ok {
			t.Fatal("Response is not a status:", resp)
		}

		if status.Tag != test.expected.Tag {
			t.Errorf("Invalid tag: expected %v but got %v", status.Tag, test.expected.Tag)
		}
		if status.Type != test.expected.Type {
			t.Errorf("Invalid type: expected %v but got %v", status.Type, test.expected.Type)
		}
		if status.Code != test.expected.Code {
			t.Errorf("Invalid code: expected %v but got %v", status.Code, test.expected.Code)
		}
		if len(status.Arguments) != len(test.expected.Arguments) {
			t.Errorf("Invalid arguments: expected %v but got %v", status.Arguments, test.expected.Arguments)
		}
		if status.Info != test.expected.Info {
			t.Errorf("Invalid info: expected %v but got %v", status.Info, test.expected.Info)
		}
	}
}

func TestParseNamedResp(t *testing.T) {
	tests := []struct {
		resp   *imap.DataResp
		name   string
		fields []interface{}
	}{
		{
			resp:   &imap.DataResp{Fields: []interface{}{"CAPABILITY", "IMAP4rev1"}},
			name:   "CAPABILITY",
			fields: []interface{}{"IMAP4rev1"},
		},
		{
			resp:   &imap.DataResp{Fields: []interface{}{"42", "EXISTS"}},
			name:   "EXISTS",
			fields: []interface{}{"42"},
		},
		{
			resp:   &imap.DataResp{Fields: []interface{}{"42", "FETCH", "blah"}},
			name:   "FETCH",
			fields: []interface{}{"42", "blah"},
		},
	}

	for _, test := range tests {
		name, fields, ok := imap.ParseNamedResp(test.resp)
		if !ok {
			t.Errorf("ParseNamedResp(%v)[2] = false, want true", test.resp)
		} else if name != test.name {
			t.Errorf("ParseNamedResp(%v)[0] = %v, want %v", test.resp, name, test.name)
		} else if !reflect.DeepEqual(fields, test.fields) {
			t.Errorf("ParseNamedResp(%v)[1] = %v, want %v", test.resp, fields, test.fields)
		}
	}
}
