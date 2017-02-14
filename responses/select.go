package responses

import (
	"fmt"

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

	mbox.Items = make(map[string]interface{})
	for h := range hdlr {
		switch res := h.Resp.(type) {
		case *imap.Resp:
			fields, ok := h.AcceptNamedResp(imap.MailboxFlags)
			if !ok {
				break
			}

			flags, _ := fields[0].([]interface{})
			mbox.Flags, _ = imap.ParseStringList(flags)
			mbox.ItemsLocker.Lock()
			mbox.Items[imap.MailboxFlags] = nil
			mbox.ItemsLocker.Unlock()
		case *imap.StatusResp:
			if len(res.Arguments) < 1 {
				h.Accepts <- false
				break
			}

			accepted := true
			switch res.Code {
			case imap.MailboxUnseen:
				mbox.Unseen, _ = imap.ParseNumber(res.Arguments[0])
				mbox.ItemsLocker.Lock()
				mbox.Items[imap.MailboxUnseen] = nil
				mbox.ItemsLocker.Unlock()
			case imap.MailboxPermanentFlags:
				flags, _ := res.Arguments[0].([]interface{})
				mbox.PermanentFlags, _ = imap.ParseStringList(flags)
				mbox.ItemsLocker.Lock()
				mbox.Items[imap.MailboxPermanentFlags] = nil
				mbox.ItemsLocker.Unlock()
			case imap.MailboxUidNext:
				mbox.UidNext, _ = imap.ParseNumber(res.Arguments[0])
				mbox.ItemsLocker.Lock()
				mbox.Items[imap.MailboxUidNext] = nil
				mbox.ItemsLocker.Unlock()
			case imap.MailboxUidValidity:
				mbox.UidValidity, _ = imap.ParseNumber(res.Arguments[0])
				mbox.ItemsLocker.Lock()
				mbox.Items[imap.MailboxUidValidity] = nil
				mbox.ItemsLocker.Unlock()
			default:
				accepted = false
			}
			h.Accepts <- accepted
		}
	}

	return
}

func (r *Select) WriteTo(w *imap.Writer) (err error) {
	status := r.Mailbox

	for k := range r.Mailbox.Items {
		switch k {
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
				Type:      imap.StatusOk,
				Code:      imap.CodePermanentFlags,
				Arguments: []interface{}{flags},
				Info:      "Flags permitted.",
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
				Type:      imap.StatusOk,
				Code:      imap.CodeUnseen,
				Arguments: []interface{}{status.Unseen},
				Info:      fmt.Sprintf("Message %d is first unseen", status.Unseen),
			}
			if err = statusRes.WriteTo(w); err != nil {
				return
			}
		case imap.MailboxUidNext:
			statusRes := &imap.StatusResp{
				Type:      imap.StatusOk,
				Code:      imap.CodeUidNext,
				Arguments: []interface{}{status.UidNext},
				Info:      "Predicted next UID",
			}
			if err = statusRes.WriteTo(w); err != nil {
				return
			}
		case imap.MailboxUidValidity:
			statusRes := &imap.StatusResp{
				Type:      imap.StatusOk,
				Code:      imap.CodeUidValidity,
				Arguments: []interface{}{status.UidValidity},
				Info:      "UIDs valid",
			}
			if err = statusRes.WriteTo(w); err != nil {
				return
			}
		}
	}

	return
}
