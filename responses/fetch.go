package responses

import (
	"github.com/emersion/go-imap"
)

// A FETCH response.
// See RFC 3501 section 7.4.2
type Fetch struct {
	Messages chan *imap.Message
}

func (r *Fetch) HandleFrom(hdlr imap.RespHandler) error {
	defer close(r.Messages)

	for h := range hdlr {
		res, ok := h.Resp.(*imap.Resp)
		if !ok || len(res.Fields) < 3 {
			h.Reject()
			continue
		}
		if name, ok := res.Fields[1].(string); !ok || name != imap.Fetch {
			h.Reject()
			continue
		}
		h.Accept()

		seqNum, _ := imap.ParseNumber(res.Fields[0])
		fields, _ := res.Fields[2].([]interface{})

		msg := &imap.Message{
			SeqNum: seqNum,
		}

		if err := msg.Parse(fields); err != nil {
			return err
		}

		r.Messages <- msg
	}

	return nil
}

func (r *Fetch) WriteTo(w *imap.Writer) error {
	for msg := range r.Messages {
		res := imap.NewUntaggedResp([]interface{}{msg.SeqNum, imap.Fetch, msg.Format()})

		if err := res.WriteTo(w); err != nil {
			return err
		}
	}

	return nil
}
