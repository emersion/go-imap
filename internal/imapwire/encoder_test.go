package imapwire

import (
	"bufio"
	"bytes"
	"testing"
)

func TestEncoderLiteral8(t *testing.T) {
	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	enc := NewEncoder(bw, ConnSideClient)

	data := []byte("he\x00llo")
	wc := enc.Literal8(int64(len(data)), nil)
	if _, err := wc.Write(data); err != nil {
		t.Fatalf("Write() = %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush() = %v", err)
	}

	// Non-synchronizing literal8 on the client side: "~{n+}\r\n" followed by
	// the raw bytes.
	want := append([]byte("~{6+}\r\n"), data...)
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("Literal8() wrote %q, want %q", buf.Bytes(), want)
	}
}
