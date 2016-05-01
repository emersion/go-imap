package common

import (
	"io"
	"strconv"
)

// A literal.
type Literal struct {
	// The literal length.
	Len int
	// The literal contents.
	Str string
}

func (l *Literal) Field() string {
	return string(literalStart) + strconv.Itoa(l.Len) + string(literalEnd)
}

// Implements io.WriterTo interface.
func (l *Literal) WriteTo(w io.Writer) (N int64, err error) {
	n, err := io.WriteString(w, l.Str)
	return int64(n), err
}
