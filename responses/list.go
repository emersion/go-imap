package responses

import (
	imap "github.com/emersion/imap/common"
)

// A LIST response.
// If Subscribed is set to true, LSUB will be used instead.
// See https://tools.ietf.org/html/rfc3501#section-7.2.2
type List struct {
	Mailboxes chan *imap.MailboxInfo
	Subscribed bool
}

func (r *List) Name() (name string) {
	name = imap.List
	if r.Subscribed {
		name = imap.Lsub
	}
	return
}

func (r *List) HandleFrom(hdlr imap.RespHandler) (err error) {
	name := r.Name()

	for h := range hdlr {
		fields, ok := h.AcceptNamedResp(name)
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

func (r *List) WriteTo(w *imap.Writer) (err error) {
	name := r.Name()

	for mbox := range r.Mailboxes {
		flags := make([]interface{}, len(mbox.Flags))
		for i, f := range mbox.Flags {
			flags[i] = f
		}

		fields := []interface{}{name, flags, mbox.Delimiter, mbox.Name}
		res := imap.NewUntaggedResp(fields)
		if err = res.WriteTo(w); err != nil {
			return
		}
	}

	return
}
