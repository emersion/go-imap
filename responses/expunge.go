package responses

import (
	imap "github.com/emersion/go-imap/common"
)

// An EXPUNGE response.
// See RFC 3501 section 7.4.1
type Expunge struct {
	SeqIds chan uint32
}

func (r *Expunge) HandleFrom(hdlr imap.RespHandler) (err error) {
	for h := range hdlr {
		res, ok := h.Resp.(*imap.Resp)
		if !ok || len(res.Fields) < 3 {
			h.Reject()
			continue
		}
		if name, ok := res.Fields[1].(string); !ok || name != imap.Expunge {
			h.Reject()
			continue
		}
		h.Accept()

		seqid, _ := imap.ParseNumber(res.Fields[0])
		r.SeqIds <- seqid
	}

	return
}

func (r *Expunge) WriteTo(w *imap.Writer) error {
	for seqid := range r.SeqIds {
		res := imap.NewUntaggedResp([]interface{}{seqid, imap.Expunge})

		if err := res.WriteTo(w); err != nil {
			return err
		}
	}

	return nil
}
