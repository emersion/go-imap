package responses

import (
	"encoding/base64"

	imap "github.com/emersion/imap/common"
)

// An AUTHENTICATE response.
type Authenticate struct {
	Mechanism imap.Sasl
	InitialResponse []byte
	Writer *imap.Writer
}

func (r *Authenticate) HandleFrom(hdlr imap.RespHandler) (err error) {
	// Cancel auth if an error occurs
	defer (func () {
		if err != nil {
			r.Writer.WriteString("*")
			r.Writer.WriteCrlf()
		}
	})()

	for h := range hdlr {
		cont, ok := h.Resp.(*imap.ContinuationResp)
		if !ok {
			h.Reject()
			continue
		}
		h.Accept()

		// Empty challenge, send initial response as stated in RFC 2222 section 5.1
		if cont.Info == "" && r.InitialResponse != nil {
			encoded := base64.StdEncoding.EncodeToString(r.InitialResponse)
			_, err = r.Writer.WriteString(encoded)
			if err != nil {
				return
			}
			_, err = r.Writer.WriteCrlf()
			if err != nil {
				return
			}

			r.InitialResponse = nil
			continue
		}

		var challenge []byte
		challenge, err = base64.StdEncoding.DecodeString(cont.Info)
		if err != nil {
			return
		}

		var res []byte
		res, err = r.Mechanism.Next(challenge)
		if err != nil {
			return
		}

		encoded := base64.StdEncoding.EncodeToString(res)
		_, err = r.Writer.WriteString(encoded)
		if err != nil {
			return
		}
		_, err = r.Writer.WriteCrlf()
		if err != nil {
			return
		}
	}

	return
}
