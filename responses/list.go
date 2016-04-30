package responses

import (
	imap "github.com/emersion/imap/common"
)

// A LIST response.
// See https://tools.ietf.org/html/rfc3501#section-7.2.2
type List struct {
	Mailboxes chan<- *imap.MailboxInfo
}

func (r *List) HandleFrom(hdlr imap.RespHandler) (err error) {
	for h := range hdlr {
		fields := h.AcceptNamedResp(imap.List)
		if fields == nil {
			continue
		}

		mbox := &imap.MailboxInfo{
			Delimiter: fields[1].(string),
			Name: fields[2].(string),
		}

		flags := fields[0].([]interface{})
		for _, f := range flags {
			mbox.Flags = append(mbox.Flags, f.(string))
		}

		r.Mailboxes <- mbox
	}

	return
}
