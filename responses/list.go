package responses

import (
	imap "github.com/emersion/imap/common"
)

// A LIST response.
// See https://tools.ietf.org/html/rfc3501#section-7.2.2
type List struct {
	Mailboxes []*imap.MailboxInfo
}

func (r *List) HandleFrom(hdlr imap.RespHandler) (err error) {
	for h := range hdlr {
		res, ok := h.Resp.(*imap.Resp)
		if !ok || getRespName(res) != imap.List {
			h.Reject()
			continue
		}
		h.Accept()

		mbox := &imap.MailboxInfo{
			Delimiter: res.Fields[2].(string),
			Name: res.Fields[3].(string),
		}

		flags := res.Fields[1].([]interface{})
		for _, f := range flags {
			mbox.Flags = append(mbox.Flags, f.(string))
		}

		r.Mailboxes = append(r.Mailboxes, mbox)
	}

	return
}
