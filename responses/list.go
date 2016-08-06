package responses

import (
	imap "github.com/emersion/go-imap/common"
)

// A LIST response.
// If Subscribed is set to true, LSUB will be used instead.
// See RFC 3501 section 7.2.2
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
		if err = mbox.Parse(fields); err != nil {
			return
		}

		r.Mailboxes <- mbox
	}

	return
}

func (r *List) WriteTo(w imap.Writer) (err error) {
	name := r.Name()

	for mbox := range r.Mailboxes {
		fields := []interface{}{name}
		fields = append(fields, mbox.Format()...)

		res := imap.NewUntaggedResp(fields)
		if err = res.WriteTo(w); err != nil {
			return
		}
	}

	return
}
