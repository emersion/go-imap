package imapclient

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func encodeFetchItems(numKind imapwire.NumKind, options *imap.FetchOptions) string {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	enc := imapwire.NewEncoder(bw, imapwire.ConnSideClient)
	writeFetchItems(enc, numKind, options)
	enc.CRLF()
	bw.Flush()
	return buf.String()
}

func TestWriteFetchItems_CustomAttributes(t *testing.T) {
	options := &imap.FetchOptions{
		UID:              true,
		Flags:            true,
		CustomAttributes: []string{"X-GM-MSGID", "X-GM-THRID", "X-GM-LABELS"},
	}
	got := encodeFetchItems(imapwire.NumKindUID, options)

	for _, want := range []string{"X-GM-MSGID", "X-GM-THRID", "X-GM-LABELS", "UID", "FLAGS"} {
		if !strings.Contains(got, want) {
			t.Errorf("encoded items missing %q: %q", want, got)
		}
	}
}

func TestOptions_CustomAttributeDecoder_CaseInsensitive(t *testing.T) {
	options := Options{
		CustomAttributeDecoders: map[string]CustomAttributeDecoderFunc{
			"X-GM-MSGID": func(d *AttributeDecoder) (CustomAttribute, error) {
				return CustomAttribute{}, nil
			},
		},
	}

	for _, name := range []string{"X-GM-MSGID", "x-gm-msgid", "X-Gm-MsgId"} {
		if got := options.customAttributeDecoder(name); got == nil {
			t.Errorf("customAttributeDecoder(%q) = nil, want non-nil", name)
		}
	}
	if got := options.customAttributeDecoder("X-GM-LABELS"); got != nil {
		t.Errorf("customAttributeDecoder(X-GM-LABELS) = %p, want nil for unregistered", got)
	}
}
