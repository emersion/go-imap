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
			if len(res.Fields) < 2 {
				h.Reject()
				break
			}

			if name, ok := res.Fields[0].(string); ok && name == "FLAGS" {
				h.Accept()

				flags := res.Fields[1].([]interface{})
				mbox.Flags = make([]string, len(flags))
				for i, f := range flags {
					mbox.Flags[i] = f.(string)
				}
			} else if name, ok := res.Fields[1].(string); ok && (name == "EXISTS" || name == "RECENT") {
				h.Accept()

				seqid := imap.ParseNumber(res.Fields[0])
				switch name {
				case "EXISTS":
					mbox.Messages = seqid
				case "RECENT":
					mbox.Recent = seqid
				}
			} else {
				h.Reject()
			}
		case *imap.StatusResp:
			accepted := true
			switch res.Code {
			case "UNSEEN":
				mbox.Unseen = imap.ParseNumber(res.Arguments[0])
			case "PERMANENTFLAGS":
				flags := res.Arguments[0].([]interface{})
				mbox.PermanentFlags = make([]string, len(flags))
				for i, f := range flags {
					mbox.PermanentFlags[i] = f.(string)
				}
			case "UIDNEXT":
				mbox.UidNext = imap.ParseNumber(res.Arguments[0])
			case "UIDVALIDITY":
				mbox.UidValidity = imap.ParseNumber(res.Arguments[0])
			default:
				accepted = false
			}
			h.Accepts <- accepted
		}
	}

	return
}
