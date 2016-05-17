package commands

import (
	"encoding/base64"
	"errors"
	"strings"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/go-sasl"
	"github.com/emersion/imap/backend"
)

// An AUTHENTICATE command.
// See RFC 3501 section 6.2.2
type Authenticate struct {
	Mechanism string
}

func (cmd *Authenticate) Command() *imap.Command {
	return &imap.Command{
		Name: imap.Authenticate,
		Arguments: []interface{}{cmd.Mechanism},
	}
}

func (cmd *Authenticate) Parse(fields []interface{}) error {
	if len(fields) < 1 {
		return errors.New("Not enough arguments")
	}

	var ok bool
	if cmd.Mechanism, ok = fields[0].(string); !ok {
		return errors.New("Mechanism must be a string")
	}

	cmd.Mechanism = strings.ToUpper(cmd.Mechanism)
	return nil
}

func (cmd *Authenticate) Handle(mechanisms map[string]sasl.ServerFactory, r *imap.Reader, w *imap.Writer) (user backend.User, err error) {
	newSasl, ok := mechanisms[cmd.Mechanism]
	if !ok {
		err = errors.New("Unsupported mechanism")
		return
	}

	sasl := newSasl()
	ir, err := sasl.Start()
	if err != nil {
		return
	}

	var encoded string
	if len(ir) > 0 {
		encoded = base64.StdEncoding.EncodeToString(ir)
	}

	cont := &imap.ContinuationResp{Info: encoded}
	if err = cont.WriteTo(w); err != nil {
		return
	}

	for {
		var encoded string
		if encoded, err = r.ReadInfo(); err != nil {
			return
		}

		var challenge []byte
		if encoded != "" {
			challenge, err = base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return
			}
		}

		var res []byte
		if res, err = sasl.Next(challenge); err != nil {
			return
		}

		// Authentication finished
		if res == nil {
			user = sasl.User()
			return
		}

		encoded = base64.StdEncoding.EncodeToString(res)
		cont := &imap.ContinuationResp{Info: encoded}
		if err = cont.WriteTo(w); err != nil {
			return
		}
	}
}
