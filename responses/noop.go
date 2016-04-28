package responses

import (
	"io"
)

// A NOOP response.
// See https://tools.ietf.org/html/rfc3501#section-6.1.2
type Noop struct {}

func (r *Noop) WriteTo(w io.Writer) (N int64, err error) {
	return
}
