package responses

import (
	"io"

	imap "github.com/emersion/imap/common"
)

// A NOOP response.
// See https://tools.ietf.org/html/rfc3501#section-6.1.2
type Noop struct {}

func (r *Noop) WriteTo(w io.Writer) (N int64, err error) {
	return
}

func ParseNoop(fields []interface{}) *Noop {
	return &Noop{}
}
