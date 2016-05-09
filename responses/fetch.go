package responses

import (
	imap "github.com/emersion/imap/common"
)

// A FETCH response.
// See https://tools.ietf.org/html/rfc3501#section-7.4.2
type Fetch struct {
	Messages chan *imap.Message
}

func (r *Fetch) HandleFrom(hdlr imap.RespHandler) (err error) {
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

		id, _ := imap.ParseNumber(res.Fields[0])
		fields, _ := res.Fields[2].([]interface{})

		msg := &imap.Message{
			Id: id,
		}

		if err = msg.Parse(fields); err != nil {
			return
		}

		r.Messages <- msg
	}

	return
}

func (r *Fetch) WriteTo(w *imap.Writer) error {
	for msg := range r.Messages {
		res := imap.NewUntaggedResp([]interface{}{msg.Id, imap.Fetch, msg.Format()})

		if err := res.WriteTo(w); err != nil {
			return err
		}
	}

	return nil
}
