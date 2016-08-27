package responses

import (
	"github.com/emersion/go-imap"
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

func (r *Search) WriteTo(w *imap.Writer) (err error) {
	fields := []interface{}{imap.Search}
	for _, id := range r.Ids {
		fields = append(fields, id)
	}

	res := imap.NewUntaggedResp(fields)
	return res.WriteTo(w)
}
