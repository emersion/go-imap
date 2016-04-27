package responses

import (
	"io"

	imap "github.com/emersion/imap/common"
)

// A STARTTLS response.
// See https://tools.ietf.org/html/rfc3501#section-6.1.2
type StartTLS struct {}

func (r *StartTLS) WriteTo(w io.Writer) (N int64, err error) {
	return
}

// TODO: add a tag parameter
func (r *StartTLS) ReadFrom(r io.Reader) (N int64, err error) {
	for {
		var resi interface{}
		resi, n, err = readResp(r)
		if err != nil {
			return
		}
		N += int64(n)

		res, ok := resi.(*imap.StatusResp)
		if !ok {
			continue
		}

		// TODO: check tag

		if res.Type != imap.OK {
			err = res
		}
		return
	}
}
