package imapwire

import (
	"bufio"
	"bytes"
	"testing"
)

func encode(f func(enc *Encoder)) string {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	enc := NewEncoder(bw, ConnSideClient)
	enc.LiteralPlus = true // so a literal never waits for a continuation in this test
	f(enc)
	bw.Flush()
	return buf.String()
}

func TestQuotedUTF8Strings(t *testing.T) {
	if got := encode(func(enc *Encoder) { enc.String("通知") }); got != "{6+}\r\n通知" {
		t.Errorf("default String = %q, want a literal", got)
	}
	got := encode(func(enc *Encoder) {
		enc.QuotedUTF8Strings = true
		enc.String(`通"知`)
	})
	if got != `"通\"知"` {
		t.Errorf("String with QuotedUTF8Strings = %q", got)
	}
	got = encode(func(enc *Encoder) {
		enc.QuotedUTF8Strings = true
		enc.Mailbox("已发送")
	})
	if got != `"&XfJT0ZAB-"` {
		t.Errorf("Mailbox with QuotedUTF8Strings = %q, want modified UTF-7", got)
	}
}
