package common

import (
	"io"
	"strconv"
)

type Literal struct {
	Len int
	Str string
}

func (l *Literal) Field() string {
	return string(literalStart) + strconv.Itoa(l.Len) + string(literalEnd)
}

func (l *Literal) WriteTo(w io.Writer) (N int64, err error) {
	n, err := io.WriteString(w, l.Str)
	return int64(n), err
}
