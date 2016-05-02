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
		fields, ok := h.AcceptNamedResp(imap.List)
		if !ok {
			continue
		}

		mbox := &imap.MailboxInfo{}

		flags, _ := fields[0].([]interface{})
		for _, f := range flags {
			if s, ok := f.(string); ok {
				mbox.Flags = append(mbox.Flags, s)
			}
		}

		mbox.Delimiter, _ = fields[1].(string)
		mbox.Name, _ = fields[2].(string)

		r.Mailboxes <- mbox
	}

	return
}
