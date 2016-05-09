package common_test

import (
	"bytes"
	"testing"

	"github.com/emersion/imap/common"
)

func TestStatusResp_WriteTo(t *testing.T) {
	tests := []struct{
		input *common.StatusResp
		expected string
	}{
		{
			input: &common.StatusResp{
				Tag: "*",
				Type: common.OK,
			},
			expected: "* OK \r\n",
		},
		{
			input: &common.StatusResp{
				Tag: "*",
				Type: common.OK,
				Info: "LOGIN completed",
			},
			expected: "* OK LOGIN completed\r\n",
		},
		{
			input: &common.StatusResp{
				Tag: "42",
				Type: common.BAD,
				Info: "Invalid arguments",
			},
			expected: "42 BAD Invalid arguments\r\n",
		},
		{
			input: &common.StatusResp{
				Tag: "a001",
				Type: common.OK,
				Code: "READ-ONLY",
				Info: "EXAMINE completed",
			},
			expected: "a001 OK [READ-ONLY] EXAMINE completed\r\n",
		},
		{
			input: &common.StatusResp{
				Tag: "*",
				Type: common.OK,
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
