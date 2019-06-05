package imap_test

import (
	"bytes"
	"testing"

	"github.com/emersion/go-imap"
)

func TestStatusResp_WriteTo(t *testing.T) {
	tests := []struct {
		input    *imap.StatusResp
		expected string
	}{
		{
			input: &imap.StatusResp{
				Tag:  "*",
				Type: imap.StatusRespOk,
			},
			expected: "* OK \r\n",
		},
		{
			input: &imap.StatusResp{
				Tag:  "*",
				Type: imap.StatusRespOk,
				Info: "LOGIN completed",
			},
			expected: "* OK LOGIN completed\r\n",
		},
		{
			input: &imap.StatusResp{
				Tag:  "42",
				Type: imap.StatusRespBad,
				Info: "Invalid arguments",
			},
			expected: "42 BAD Invalid arguments\r\n",
		},
		{
			input: &imap.StatusResp{
				Tag:  "a001",
				Type: imap.StatusRespOk,
				Code: "READ-ONLY",
				Info: "EXAMINE completed",
			},
			expected: "a001 OK [READ-ONLY] EXAMINE completed\r\n",
		},
		{
			input: &imap.StatusResp{
				Tag:       "*",
				Type:      imap.StatusRespOk,
				Code:      "CAPABILITY",
				Arguments: []interface{}{imap.RawString("IMAP4rev1")},
				Info:      "IMAP4rev1 service ready",
			},
			expected: "* OK [CAPABILITY IMAP4rev1] IMAP4rev1 service ready\r\n",
		},
	}

	for i, test := range tests {
		b := &bytes.Buffer{}
		w := imap.NewWriter(b)

		if err := test.input.WriteTo(w); err != nil {
			t.Errorf("Cannot write status #%v, got error: %v", i, err)
			continue
		}

		o := b.String()
		if o != test.expected {
			t.Errorf("Invalid output for status #%v: %v", i, o)
		}
	}
}

func TestStatus_Err(t *testing.T) {
	status := &imap.StatusResp{Type: imap.StatusRespOk, Info: "All green"}
	if err := status.Err(); err != nil {
		t.Error("OK status returned error:", err)
	}

	status = &imap.StatusResp{Type: imap.StatusRespBad, Info: "BAD!"}
	if err := status.Err(); err == nil {
		t.Error("BAD status didn't returned error:", err)
	} else if err.Error() != "BAD!" {
		t.Error("BAD status returned incorrect error message:", err)
	}

	status = &imap.StatusResp{Type: imap.StatusRespNo, Info: "NO!"}
	if err := status.Err(); err == nil {
		t.Error("NO status didn't returned error:", err)
	} else if err.Error() != "NO!" {
		t.Error("NO status returned incorrect error message:", err)
	}
}
