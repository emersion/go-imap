package responses

import (
	"errors"

	imap "github.com/emersion/imap/common"
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

	for h := range hdlr {
		fields, ok := h.AcceptNamedResp(imap.Status)
		if !ok {
			continue
		}
		if len(fields) < 2 {
			return errors.New("STATUS response excepts two fields")
		}

		if mbox.Name, ok = fields[0].(string); !ok {
			return errors.New("STATUS response excepts a string as first argument")
		}

		var items []interface{}
		if items, ok = fields[1].([]interface{}); !ok {
			return errors.New("STATUS response excepts a list as second argument")
		}

		var key string
		for i, f := range items {
			if i % 2 == 0 {
				var ok bool
				if key, ok = f.(string); !ok {
					return errors.New("Key is not a string")
				}
			} else {
				switch key {
				case "MESSAGES":
					mbox.Messages = imap.ParseNumber(f)
				case "RECENT":
					mbox.Recent = imap.ParseNumber(f)
				case "UIDNEXT":
					mbox.UidNext = imap.ParseNumber(f)
				case "UIDVALIDITY":
					mbox.UidValidity = imap.ParseNumber(f)
				case "UNSEEN":
					mbox.Unseen = imap.ParseNumber(f)
				}
			}
		}
	}

	return nil
}
