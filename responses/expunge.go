package responses

import (
	imap "github.com/emersion/go-imap/common"
)

// An EXPUNGE response.
// See RFC 3501 section 7.4.1
type Expunge struct {
	SeqNums chan uint32
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

		seqNum, _ := imap.ParseNumber(res.Fields[0])
		r.SeqNums <- seqNum
	}

	return
}

func (r *Expunge) WriteTo(w imap.Writer) error {
	for seqNum := range r.SeqNums {
		res := imap.NewUntaggedResp([]interface{}{seqNum, imap.Expunge})

		if err := res.WriteTo(w); err != nil {
			return err
		}
	}

	return nil
}
