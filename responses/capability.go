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

func ParseCapability(fields []interface{}) *Capability {
	caps := make([]string, len(fields))

	for i, c := range fields {
		caps[i] = c.(string)
	}

	return &Capability{
		Capabilities: caps,
	}
}
