package common_test

import (
	"bytes"
	"testing"

	"github.com/emersion/go-imap/common"
)

func TestStatusResp_WriteTo(t *testing.T) {
	tests := []struct{
		input *common.StatusResp
		expected string
	}{
		{
			input: &common.StatusResp{
				Tag: "*",
				Type: common.StatusOk,
			},
			expected: "* OK \r\n",
		},
		{
			input: &common.StatusResp{
				Tag: "*",
				Type: common.StatusOk,
				Info: "LOGIN completed",
			},
			expected: "* OK LOGIN completed\r\n",
		},
		{
			input: &common.StatusResp{
				Tag: "42",
				Type: common.StatusBad,
				Info: "Invalid arguments",
			},
			expected: "42 BAD Invalid arguments\r\n",
		},
		{
			input: &common.StatusResp{
				Tag: "a001",
				Type: common.StatusOk,
				Code: "READ-ONLY",
				Info: "EXAMINE completed",
			},
			expected: "a001 OK [READ-ONLY] EXAMINE completed\r\n",
		},
		{
			input: &common.StatusResp{
				Tag: "*",
				Type: common.StatusOk,
				Code: "CAPABILITY",
				Arguments: []interface{}{"IMAP4rev1"},
				Info: "IMAP4rev1 service ready",
			},
			expected: "* OK [CAPABILITY IMAP4rev1] IMAP4rev1 service ready\r\n",
		},
	}

	for i, test := range tests {
		b := &bytes.Buffer{}
		w := common.NewWriter(b)

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
	status := &common.StatusResp{Type: common.StatusOk, Info: "All green"}
	if err := status.Err(); err != nil {
		t.Error("OK status returned error:", err)
	}

	status = &common.StatusResp{Type: common.StatusBad, Info: "BAD!"}
	if err := status.Err(); err == nil {
		t.Error("BAD status didn't returned error:", err)
	} else if err.Error() != "BAD!" {
		t.Error("BAD status returned incorrect error message:", err)
	}

	status = &common.StatusResp{Type: common.StatusNo, Info: "NO!"}
	if err := status.Err(); err == nil {
		t.Error("NO status didn't returned error:", err)
	} else if err.Error() != "NO!" {
		t.Error("NO status returned incorrect error message:", err)
	}
}
