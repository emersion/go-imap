package responses

import (
	"fmt"

	"github.com/emersion/go-imap"
)

// A SELECT response.
type Select struct {
	Mailbox *imap.MailboxStatus
}

func (r *Select) Handle(resp imap.Resp) error {
	if r.Mailbox == nil {
		r.Mailbox = &imap.MailboxStatus{Items: make(map[string]interface{})}
	}
	mbox := r.Mailbox

	switch resp := resp.(type) {
	case *imap.DataResp:
		name, fields, ok := imap.ParseNamedResp(resp)
		if !ok || name != imap.MailboxFlags {
			return ErrUnhandled
		} else if len(fields) < 1 {
			return errNotEnoughFields
		}

		flags, _ := fields[0].([]interface{})
		mbox.Flags, _ = imap.ParseStringList(flags)
		mbox.ItemsLocker.Lock()
		mbox.Items[imap.MailboxFlags] = nil
		mbox.ItemsLocker.Unlock()
	case *imap.StatusResp:
		if len(resp.Arguments) < 1 {
			return ErrUnhandled
		}

		switch resp.Code {
		case imap.MailboxUnseen:
			mbox.Unseen, _ = imap.ParseNumber(resp.Arguments[0])
		case imap.MailboxPermanentFlags:
			flags, _ := resp.Arguments[0].([]interface{})
			mbox.PermanentFlags, _ = imap.ParseStringList(flags)
		case imap.MailboxUidNext:
			mbox.UidNext, _ = imap.ParseNumber(resp.Arguments[0])
		case imap.MailboxUidValidity:
			mbox.UidValidity, _ = imap.ParseNumber(resp.Arguments[0])
		default:
			return ErrUnhandled
		}

		mbox.ItemsLocker.Lock()
		mbox.Items[resp.Code] = nil
		mbox.ItemsLocker.Unlock()
	default:
		return ErrUnhandled
	}
	return nil
}

func (r *Select) WriteTo(w *imap.Writer) error {
	mbox := r.Mailbox

	for k := range r.Mailbox.Items {
		switch k {
		case imap.MailboxFlags:
			flags := make([]interface{}, len(mbox.Flags))
			for i, f := range mbox.Flags {
				flags[i] = f
			}
			res := imap.NewUntaggedResp([]interface{}{"FLAGS", flags})
			if err := res.WriteTo(w); err != nil {
				return err
			}
		case imap.MailboxPermanentFlags:
			flags := make([]interface{}, len(mbox.PermanentFlags))
			for i, f := range mbox.PermanentFlags {
				flags[i] = f
			}
			statusRes := &imap.StatusResp{
				Type:      imap.StatusOk,
				Code:      imap.CodePermanentFlags,
				Arguments: []interface{}{flags},
				Info:      "Flags permitted.",
			}
			if err := statusRes.WriteTo(w); err != nil {
				return err
			}
		case imap.MailboxMessages:
			res := imap.NewUntaggedResp([]interface{}{mbox.Messages, "EXISTS"})
			if err := res.WriteTo(w); err != nil {
				return err
			}
		case imap.MailboxRecent:
			res := imap.NewUntaggedResp([]interface{}{mbox.Recent, "RECENT"})
			if err := res.WriteTo(w); err != nil {
				return err
			}
		case imap.MailboxUnseen:
			statusRes := &imap.StatusResp{
				Type:      imap.StatusOk,
				Code:      imap.CodeUnseen,
				Arguments: []interface{}{mbox.Unseen},
				Info:      fmt.Sprintf("Message %d is first unseen", mbox.Unseen),
			}
			if err := statusRes.WriteTo(w); err != nil {
				return err
			}
		case imap.MailboxUidNext:
			statusRes := &imap.StatusResp{
				Type:      imap.StatusOk,
				Code:      imap.CodeUidNext,
				Arguments: []interface{}{mbox.UidNext},
				Info:      "Predicted next UID",
			}
			if err := statusRes.WriteTo(w); err != nil {
				return err
			}
		case imap.MailboxUidValidity:
			statusRes := &imap.StatusResp{
				Type:      imap.StatusOk,
				Code:      imap.CodeUidValidity,
				Arguments: []interface{}{mbox.UidValidity},
				Info:      "UIDs valid",
			}
			if err := statusRes.WriteTo(w); err != nil {
				return err
			}
		}
	}

	return nil
}
