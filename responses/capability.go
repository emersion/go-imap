package responses

import (
	"io"

	imap "github.com/emersion/imap/common"
)

// A CAPABILITY response.
// See https://tools.ietf.org/html/rfc3501#section-7.2.1
type Capability struct {
	Capabilities []string
}

func (r *Capability) WriteTo(w io.Writer) (N int64, err error) {
	// Insert IMAP4rev1 at the begining of capabilities list
	caps := []string{"IMAP4rev1"}
	caps = append(caps, r.Capabilities...)

	res := &imap.Response{
		Tag: imap.Capability,
		Fields: caps,
	}

	return res.WriteTo(w)
}

// TODO: add a tag parameter
func (r *Capability) ReadFrom(r io.Reader) (N int64, err error) {
	for {
		var resi interface{}
		resi, n, err = readResp(r)
		if err != nil {
			return
		}
		N += int64(n)

		switch res := resi.(type) {
		case *imap.Response:
			name := res.Fields[0].(string)
			if name != imap.Capability {
				continue
			}

			caps := res.Fields[1:]
			r.Capabilities = make([]string, len(caps))
			for i, c := caps {
				r.Capabilities[i] = c.(string)
			}
		case *imap.StatusResp:
			// TODO: check tag
			if res.Type != imap.OK {
				err = res
			}
			return
		}
	}
}
