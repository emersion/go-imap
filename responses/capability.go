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
	caps := []interface{}{"IMAP4rev1"}
	for _, c := range r.Capabilities {
		caps = append(caps, c)
	}

	res := &imap.Resp{
		Tag: imap.Capability,
		Fields: caps,
	}

	return res.WriteTo(w)
}

func (r *Capability) HandleFrom(hdlr imap.RespHandler) (err error) {
	for h := range hdlr {
		res, ok := h.Resp.(*imap.Resp)
		if !ok || len(res.Fields) == 0 {
			h.Reject()
			continue
		}

		name, ok := res.Fields[0].(string)
		if !ok || name != imap.Capability {
			h.Reject()
			continue
		}

		h.Accept()

		caps := res.Fields[1:]
		r.Capabilities = make([]string, len(caps))
		for i, c := range caps {
			r.Capabilities[i] = c.(string)
		}
	}

	return
}
