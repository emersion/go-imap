package responses

import (
	"io"
)

// A STARTTLS response.
// See https://tools.ietf.org/html/rfc3501#section-6.1.2
type StartTLS struct {}

func (r *StartTLS) WriteTo(w io.Writer) (N int64, err error) {
	return
}
