package responses

import (
	"github.com/emersion/go-imap"
)

// A LIST response.
// If Subscribed is set to true, LSUB will be used instead.
// See RFC 3501 section 7.2.2
type List struct {
	Mailboxes  chan *imap.MailboxInfo
	Subscribed bool
}

func (r *List) Name() string {
	if r.Subscribed {
		return imap.Lsub
	} else {
		return imap.List
	}
}

func (r *List) HandleFrom(hdlr imap.RespHandler) error {
	defer close(r.Mailboxes)

	name := r.Name()

	for h := range hdlr {
		fields, ok := h.AcceptNamedResp(name)
		if !ok {
			continue
		}

		mbox := &imap.MailboxInfo{}
		if err := mbox.Parse(fields); err != nil {
			return err
		}

		r.Mailboxes <- mbox
	}

	return nil
}

func (r *List) WriteTo(w *imap.Writer) error {
	name := r.Name()

	for mbox := range r.Mailboxes {
		fields := []interface{}{name}
		fields = append(fields, mbox.Format()...)

		res := imap.NewUntaggedResp(fields)
		if err := res.WriteTo(w); err != nil {
			return err
		}
	}

	return nil
}
