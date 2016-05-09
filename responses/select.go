package responses

import (
	imap "github.com/emersion/imap/common"
)

// A SELECT response.
type Select struct {
	Mailbox *imap.MailboxStatus
}

func (r *Select) HandleFrom(hdlr imap.RespHandler) (err error) {
	if r.Mailbox == nil {
		r.Mailbox = &imap.MailboxStatus{}
	}
	mbox := r.Mailbox

	for h := range hdlr {
		switch res := h.Resp.(type) {
		case *imap.Resp:
			fields, ok := h.AcceptNamedResp("FLAGS")
			if !ok {
				break
			}

			flags, _ := fields[0].([]interface{})
			mbox.Flags = make([]string, len(flags))
			for i, f := range flags {
				mbox.Flags[i], _ = f.(string)
			}
		case *imap.StatusResp:
			accepted := true
			switch res.Code {
			case "UNSEEN":
				mbox.Unseen, _ = imap.ParseNumber(res.Arguments[0])
			case "PERMANENTFLAGS":
				flags, _ := res.Arguments[0].([]interface{})
				mbox.PermanentFlags = make([]string, len(flags))
				for i, f := range flags {
					mbox.PermanentFlags[i], _ = f.(string)
				}
			case "UIDNEXT":
				mbox.UidNext, _ = imap.ParseNumber(res.Arguments[0])
			case "UIDVALIDITY":
				mbox.UidValidity, _ = imap.ParseNumber(res.Arguments[0])
			default:
				accepted = false
			}
			h.Accepts <- accepted
		}
	}

	return
}
