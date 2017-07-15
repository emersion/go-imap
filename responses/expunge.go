package responses

import (
	"github.com/emersion/go-imap"
)

// An EXPUNGE response.
// See RFC 3501 section 7.4.1
type Expunge struct {
	SeqNums chan uint32
}

func (r *Expunge) Handle(resp imap.Resp) error {
	name, fields, ok := imap.ParseNamedResp(resp)
	if !ok || name != imap.Expunge {
		return ErrUnhandled
	}

	if len(fields) == 0 {
		return errNotEnoughFields
	}

	seqNum, err := imap.ParseNumber(fields[0])
	if err != nil {
		return err
	}

	r.SeqNums <- seqNum
	return nil
}

func (r *Expunge) WriteTo(w *imap.Writer) error {
	for seqNum := range r.SeqNums {
		resp := imap.NewUntaggedResp([]interface{}{seqNum, imap.Expunge})
		if err := resp.WriteTo(w); err != nil {
			return err
		}
	}

	return nil
}
