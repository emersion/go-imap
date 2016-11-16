package responses

import (
	"encoding/base64"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-sasl"
)

// An AUTHENTICATE response.
type Authenticate struct {
	Mechanism       sasl.Client
	InitialResponse []byte
	Writer          *imap.Writer
}

func (r *Authenticate) HandleFrom(hdlr imap.RespHandler) (err error) {
	w := r.Writer

	// Cancel auth if an error occurs
	defer (func() {
		if err != nil {
			w.Write([]byte("*\r\n"))
			w.Flush()
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
			if _, err = w.Write([]byte(encoded + "\r\n")); err != nil {
				return
			}
			if err = w.Flush(); err != nil {
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
		if _, err = w.Write([]byte(encoded + "\r\n")); err != nil {
			return
		}
		if err = w.Flush(); err != nil {
			return
		}
	}

	return
}
