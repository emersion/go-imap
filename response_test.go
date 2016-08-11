package imap_test

import (
	"bytes"
	"testing"

	"github.com/emersion/go-imap"
)

func TestResp_WriteTo(t *testing.T) {
	var b bytes.Buffer
	w := imap.NewWriter(&b)

	resp := &imap.Resp{
		Fields: []interface{}{"76", "FETCH", []interface{}{"UID", 783}},
	}

	if err := resp.WriteTo(w); err != nil {
		t.Fatal(err)
	}

	if b.String() != "* 76 FETCH (UID 783)\r\n" {
		t.Error("Invalid response:", b.String())
	}
}

func TestContinuationResp_WriteTo(t *testing.T) {
	var b bytes.Buffer
	w := imap.NewWriter(&b)

	resp := &imap.ContinuationResp{}

	if err := resp.WriteTo(w); err != nil {
		t.Fatal(err)
	}

	if b.String() != "+\r\n" {
		t.Error("Invalid continuation:", b.String())
	}
}

func TestContinuationResp_WriteTo_WithInfo(t *testing.T) {
	var b bytes.Buffer
	w := imap.NewWriter(&b)

	resp := &imap.ContinuationResp{Info: "send literal"}

	if err := resp.WriteTo(w); err != nil {
		t.Fatal(err)
	}

	if b.String() != "+ send literal\r\n" {
		t.Error("Invalid continuation:", b.String())
	}
}

func TestReadResp_ContinuationResp(t *testing.T) {
	b := bytes.NewBufferString("+ send literal\r\n")
	r := imap.NewReader(b)

	resp, err := imap.ReadResp(r)
	if err != nil {
		t.Fatal(err)
	}

	cont, ok := resp.(*imap.ContinuationResp)
	if !ok {
		t.Fatal("Response is not a continuation request")
	}

	if cont.Info != "send literal" {
		t.Error("Invalid info:", cont.Info)
	}
}

func TestReadResp_ContinuationResp_NoInfo(t *testing.T) {
	b := bytes.NewBufferString("+\r\n")
	r := imap.NewReader(b)

	resp, err := imap.ReadResp(r)
	if err != nil {
		t.Fatal(err)
	}

	cont, ok := resp.(*imap.ContinuationResp)
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

	respi, err := imap.ReadResp(r)
	if err != nil {
		t.Fatal(err)
	}

	resp, ok := respi.(*imap.Resp)
	if !ok {
		t.Fatal("Invalid response type")
	}

	if resp.Tag != "*" {
		t.Error("Invalid tag:", resp.Tag)
	}
	if len(resp.Fields) != 2 {
		t.Error("Invalid fields:", resp.Fields)
	}
}

func TestReadResp_Resp_NoArgs(t *testing.T) {
	b := bytes.NewBufferString("* SEARCH\r\n")
	r := imap.NewReader(b)

	respi, err := imap.ReadResp(r)
	if err != nil {
		t.Fatal(err)
	}

	resp, ok := respi.(*imap.Resp)
	if !ok {
		t.Fatal("Invalid response type")
	}

	if resp.Tag != "*" {
		t.Error("Invalid tag:", resp.Tag)
	}
	if len(resp.Fields) != 1 || resp.Fields[0] != "SEARCH" {
		t.Error("Invalid fields:", resp.Fields)
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
				Type: imap.StatusOk,
				Info: "IMAP4rev1 Service Ready",
			},
		},
		{
			input: "* PREAUTH Welcome Pauline!\r\n",
			expected: &imap.StatusResp{
				Tag:  "*",
				Type: imap.StatusPreauth,
				Info: "Welcome Pauline!",
			},
		},
		{
			input: "a001 OK NOOP completed\r\n",
			expected: &imap.StatusResp{
				Tag:  "a001",
				Type: imap.StatusOk,
				Info: "NOOP completed",
			},
		},
		{
			input: "a001 OK [READ-ONLY] SELECT completed\r\n",
			expected: &imap.StatusResp{
				Tag:  "a001",
				Type: imap.StatusOk,
				Code: "READ-ONLY",
				Info: "SELECT completed",
			},
		},
		{
			input: "a001 OK [CAPABILITY IMAP4rev1 UIDPLUS] LOGIN completed\r\n",
			expected: &imap.StatusResp{
				Tag:       "a001",
				Type:      imap.StatusOk,
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
