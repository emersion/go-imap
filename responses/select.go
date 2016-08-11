package responses

import (
	"github.com/emersion/go-imap"
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
			mbox.Items = append(mbox.Items, imap.MailboxFlags)
		case *imap.StatusResp:
			accepted := true
			switch res.Code {
			case "UNSEEN":
				mbox.Unseen, _ = imap.ParseNumber(res.Arguments[0])
				mbox.Items = append(mbox.Items, imap.MailboxUnseen)
			case "PERMANENTFLAGS":
				flags, _ := res.Arguments[0].([]interface{})
				mbox.PermanentFlags = make([]string, len(flags))
				for i, f := range flags {
					mbox.PermanentFlags[i], _ = f.(string)
				}
				mbox.Items = append(mbox.Items, imap.MailboxPermanentFlags)
			case "UIDNEXT":
				mbox.UidNext, _ = imap.ParseNumber(res.Arguments[0])
				mbox.Items = append(mbox.Items, imap.MailboxUidNext)
			case "UIDVALIDITY":
				mbox.UidValidity, _ = imap.ParseNumber(res.Arguments[0])
				mbox.Items = append(mbox.Items, imap.MailboxUidValidity)
			default:
				accepted = false
			}
			h.Accepts <- accepted
		}
	}

	return
}

func (r *Select) WriteTo(w imap.Writer) (err error) {
	status := r.Mailbox

	for _, item := range r.Mailbox.Items {
		switch item {
		case imap.MailboxFlags:
			flags := make([]interface{}, len(status.Flags))
			for i, f := range status.Flags {
				flags[i] = f
			}
			res := imap.NewUntaggedResp([]interface{}{"FLAGS", flags})
			if err = res.WriteTo(w); err != nil {
				return
			}
		case imap.MailboxPermanentFlags:
			flags := make([]interface{}, len(status.PermanentFlags))
			for i, f := range status.PermanentFlags {
				flags[i] = f
			}
			statusRes := &imap.StatusResp{
				Type: imap.StatusOk,
				Code: imap.CodePermanentFlags,
				Arguments: []interface{}{flags},
				Info: "Flags permitted.",
			}
			if err = statusRes.WriteTo(w); err != nil {
				return
			}
		case imap.MailboxMessages:
			res := imap.NewUntaggedResp([]interface{}{status.Messages, "EXISTS"})
			if err = res.WriteTo(w); err != nil {
				return
			}
		case imap.MailboxRecent:
			res := imap.NewUntaggedResp([]interface{}{status.Recent, "RECENT"})
			if err = res.WriteTo(w); err != nil {
				return
			}
		case imap.MailboxUnseen:
			statusRes := &imap.StatusResp{
				Type: imap.StatusOk,
				Code: imap.CodeUnseen,
				Arguments: []interface{}{status.Unseen},
			}
			if err = statusRes.WriteTo(w); err != nil {
				return
			}
		case imap.MailboxUidNext:
			statusRes := &imap.StatusResp{
				Type: imap.StatusOk,
				Code: imap.CodeUidNext,
				Arguments: []interface{}{status.UidNext},
				Info: "Predicted next UID",
			}
			if err = statusRes.WriteTo(w); err != nil {
				return
			}
		case imap.MailboxUidValidity:
			statusRes := &imap.StatusResp{
				Type: imap.StatusOk,
				Code: imap.CodeUidValidity,
				Arguments: []interface{}{status.UidValidity},
				Info: "UIDs valid",
			}
			if err = statusRes.WriteTo(w); err != nil {
				return
			}
		}
	}

	return
}
