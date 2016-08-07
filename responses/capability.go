package responses

import (
	imap "github.com/emersion/go-imap/common"
)

// A CAPABILITY response.
// See RFC 3501 section 7.2.1
type Capability struct {
	Caps []string
}

func (r *Capability) HandleFrom(hdlr imap.RespHandler) (err error) {
	for h := range hdlr {
		caps, ok := h.AcceptNamedResp(imap.Capability)
		if !ok {
			continue
		}

		r.Caps = make([]string, len(caps))
		for i, c := range caps {
			r.Caps[i], _ = c.(string)
		}

		return
	}

	return
}

func (r *Capability) WriteTo(w imap.Writer) error {
	fields := []interface{}{imap.Capability}
	for _, cap := range r.Caps {
		fields = append(fields, cap)
	}

	res := &imap.Resp{Fields: fields}
	return res.WriteTo(w)
}
