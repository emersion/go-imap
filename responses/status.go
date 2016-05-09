package responses

import (
	"errors"
	"strings"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/utf7"
)

// A STATUS response.
// See https://tools.ietf.org/html/rfc3501#section-7.2.4
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

		var items []interface{}
		if items, ok = fields[1].([]interface{}); !ok {
			return errors.New("STATUS response expects a list as second argument")
		}

		var key string
		for i, f := range items {
			if i % 2 == 0 {
				var ok bool
				if key, ok = f.(string); !ok {
					return errors.New("Key is not a string")
				}
			} else {
				key = strings.ToUpper(key)
				mbox.Items = append(mbox.Items, key)

				switch key {
				case "MESSAGES":
					mbox.Messages, _ = imap.ParseNumber(f)
				case "RECENT":
					mbox.Recent, _ = imap.ParseNumber(f)
				case "UIDNEXT":
					mbox.UidNext, _ = imap.ParseNumber(f)
				case "UIDVALIDITY":
					mbox.UidValidity, _ = imap.ParseNumber(f)
				case "UNSEEN":
					mbox.Unseen, _ = imap.ParseNumber(f)
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
		case "MESSAGES":
			value = mbox.Messages
		case "RECENT":
			value = mbox.Recent
		case "UIDNEXT":
			value = mbox.UidNext
		case "UIDVALIDITY":
			value = mbox.UidValidity
		case "UNSEEN":
			value = mbox.Unseen
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
