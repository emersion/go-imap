package responses

import (
	"errors"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// A STATUS response.
// See RFC 3501 section 7.2.4
type Status struct {
	Mailbox *imap.MailboxStatus
}

func (r *Status) HandleFrom(hdlr imap.RespHandler) error {
	if r.Mailbox == nil {
		r.Mailbox = &imap.MailboxStatus{}
	}
	mbox := r.Mailbox
	mbox.Items = nil

	for h := range hdlr {
		fields, ok := h.AcceptNamedResp(imap.Status)
		if !ok {
			continue
		}
		if len(fields) < 2 {
			return errors.New("STATUS response expects two fields")
		}

		name, ok := fields[0].(string)
		if !ok {
			return errors.New("STATUS response expects a string as first argument")
		}
		mbox.Name, _ = utf7.Decoder.String(name)
		mbox.Name = imap.CanonicalMailboxName(mbox.Name)

		var items []interface{}
		if items, ok = fields[1].([]interface{}); !ok {
			return errors.New("STATUS response expects a list as second argument")
		}

		if err := mbox.Parse(items); err != nil {
			return err
		}
	}

	return nil
}

func (r *Status) WriteTo(w *imap.Writer) error {
	mbox := r.Mailbox

	fields := []interface{}{imap.Status, mbox.Name, mbox.Format()}

	res := imap.NewUntaggedResp(fields)
	if err := res.WriteTo(w); err != nil {
		return err
	}

	return nil
}
