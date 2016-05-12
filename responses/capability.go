package responses

import (
	imap "github.com/emersion/imap/common"
)

// A CAPABILITY response.
// See RFC 3501 section 7.2.1
type Capability struct {
	Caps []string
}

func (r *Capability) Response() *imap.Resp {
	fields := []interface{}{imap.Capability}
	for _, cap := range r.Caps {
		fields = append(fields, cap)
	}

	return &imap.Resp{
		Tag: "*",
		Fields: fields,
	}
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
