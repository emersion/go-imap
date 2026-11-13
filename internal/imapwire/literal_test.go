package imapwire

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"
)

func TestLiteralReaderTerminalError(t *testing.T) {
	wrappedEOF := fmt.Errorf("read failed: %w", io.EOF)
	tests := []struct {
		name    string
		input   string
		readErr error
		body    string
		wantErr error
	}{
		{name: "complete_literal", input: "{5}\r\nhello)", body: "hello"},
		{name: "empty_literal", input: "{0}\r\n)", body: ""},
		{name: "quoted_string", input: "\"hello\")", body: "hello"},
		{name: "short_literal", input: "{8}\r\nshort", body: "short", wantErr: io.ErrUnexpectedEOF},
		{name: "missing_literal", input: "{8}\r\n", wantErr: io.ErrUnexpectedEOF},
		{name: "read_timeout", input: "{8}\r\nshort", readErr: os.ErrDeadlineExceeded, body: "short", wantErr: os.ErrDeadlineExceeded},
		{name: "closed_pipe", input: "{8}\r\nshort", readErr: io.ErrClosedPipe, body: "short", wantErr: io.ErrClosedPipe},
		{name: "wrapped_eof", input: "{8}\r\nshort", readErr: wrappedEOF, body: "short", wantErr: wrappedEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var source io.Reader = strings.NewReader(tt.input)
			if tt.readErr != nil {
				source = io.MultiReader(source, iotest.ErrReader(tt.readErr))
			}
			dec := NewDecoder(bufio.NewReader(source), ConnSideClient)
			lit, _, ok := dec.ExpectNStringReader()
			if !ok || lit == nil {
				t.Fatalf("ExpectNStringReader() = %v, %v; decoder error: %v", lit, ok, dec.Err())
			}
			counter := &literalReadCounter{Reader: lit.r}
			lit.r = counter
			body, err := io.ReadAll(lit)
			if string(body) != tt.body || !errors.Is(err, tt.wantErr) {
				t.Fatalf("ReadAll() = %q, %v; want %q, %v", body, err, tt.body, tt.wantErr)
			}
			if !errors.Is(dec.Err(), tt.wantErr) {
				t.Errorf("decoder error = %v; want %v", dec.Err(), tt.wantErr)
			}
			if dec.literal {
				t.Error("literal remains open after its terminal read")
			}
			calls := counter.calls
			wantTerminalErr := tt.wantErr
			if wantTerminalErr == nil {
				wantTerminalErr = io.EOF
			}
			for i := 0; i < 2; i++ {
				n, err := lit.Read(make([]byte, 1))
				if n != 0 || !errors.Is(err, wantTerminalErr) {
					t.Fatalf("Read() after completion = %v, %v; want 0, %v", n, err, wantTerminalErr)
				}
			}
			if counter.calls != calls {
				t.Errorf("underlying reader called after completion: %v -> %v", calls, counter.calls)
			}
			if tt.wantErr == nil && !dec.ExpectSpecial(')') {
				t.Errorf("cannot decode the next field: %v", dec.Err())
			}
		})
	}
}

type literalReadCounter struct {
	io.Reader
	calls int
}

func (r *literalReadCounter) Read(p []byte) (int, error) {
	r.calls++
	return r.Reader.Read(p)
}
