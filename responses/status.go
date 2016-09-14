package responses

import (
	"errors"
	"strings"

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

		var key string
		for i, f := range items {
			if i%2 == 0 {
				var ok bool
				if key, ok = f.(string); !ok {
					return errors.New("Key is not a string")
				}
			} else {
				key = strings.ToUpper(key)
				mbox.Items = append(mbox.Items, key)

				switch key {
				case imap.MailboxMessages:
					mbox.Messages, _ = imap.ParseNumber(f)
				case imap.MailboxRecent:
					mbox.Recent, _ = imap.ParseNumber(f)
				case imap.MailboxUnseen:
					mbox.Unseen, _ = imap.ParseNumber(f)
				case imap.MailboxUidNext:
					mbox.UidNext, _ = imap.ParseNumber(f)
				case imap.MailboxUidValidity:
					mbox.UidValidity, _ = imap.ParseNumber(f)
				}
			}
		}
	}

	return nil
}

func (r *Status) WriteTo(w *imap.Writer) error {
	mbox := r.Mailbox

	var fields []interface{}
	for _, item := range mbox.Items {
		var value interface{}
		switch strings.ToUpper(item) {
		case imap.MailboxMessages:
			value = mbox.Messages
		case imap.MailboxRecent:
			value = mbox.Recent
		case imap.MailboxUnseen:
			value = mbox.Unseen
		case imap.MailboxUidNext:
			value = mbox.UidNext
		case imap.MailboxUidValidity:
			value = mbox.UidValidity
		}

		fields = append(fields, item, value)
	}

	name, _ := utf7.Encoder.String(mbox.Name)

	fields = append([]interface{}{imap.Status, name}, fields)
	res := imap.NewUntaggedResp(fields)
	if err := res.WriteTo(w); err != nil {
		return err
	}

	return nil
}
