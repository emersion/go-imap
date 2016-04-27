package responses

import (
	"io"

	imap "github.com/emersion/imap/common"
)

// A STARTTLS response.
// See https://tools.ietf.org/html/rfc3501#section-6.1.2
type Starttls struct {}

func (r *Starttls) WriteTo(w io.Writer) (N int64, err error) {
	return
}
