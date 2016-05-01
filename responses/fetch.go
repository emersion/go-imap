package responses

import (
	imap "github.com/emersion/imap/common"
)

// A FETCH response.
// See https://tools.ietf.org/html/rfc3501#section-7.4.2
type Fetch struct {
	Messages chan<- *imap.Message
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

		id := imap.ParseNumber(res.Fields[0])
		fields := res.Fields[2].([]interface{})

		msg := &imap.Message{
			Id: id,
			Fields: map[string]interface{}{},
		}

		var key string
		for i, f := range fields {
			if i % 2 == 0 {// It's a key
				key = f.(string)
			} else { // It's a value
				msg.Fields[key] = f
			}
		}

		r.Messages <- msg
	}

	return
}
