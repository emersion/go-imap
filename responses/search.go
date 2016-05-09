package responses

import (
	imap "github.com/emersion/imap/common"
)

// A SEARCH response.
// See RFC 3501 section 7.2.5
type Search struct {
	Ids []uint32
}

func (r *Search) HandleFrom(hdlr imap.RespHandler) (err error) {
	for h := range hdlr {
		fields, ok := h.AcceptNamedResp(imap.Search)
		if !ok {
			continue
		}

		for _, f := range fields {
			id, _ := imap.ParseNumber(f)
			r.Ids = append(r.Ids, id)
		}
	}

	return
}
