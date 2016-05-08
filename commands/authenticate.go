package commands

import (
	"encoding/base64"
	"errors"
	"strings"

	imap "github.com/emersion/imap/common"
)

// An AUTHENTICATE command.
// See https://tools.ietf.org/html/rfc3501#section-6.2.2
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

func (cmd *Authenticate) Handle(mechanisms map[string]imap.SaslServer, r *imap.Reader, w *imap.Writer) error {
	sasl, ok := mechanisms[cmd.Mechanism]
	if !ok {
		return errors.New("Unsupported mechanism")
	}

	ir, err := sasl.Start()
	if err != nil {
		return err
	}

	var encoded string
	if len(ir) > 0 {
		encoded = base64.StdEncoding.EncodeToString(ir)
	}

	cont := &imap.ContinuationResp{Info: encoded}
	if err := cont.WriteTo(w); err != nil {
		return err
	}

	for {
		encoded, err := r.ReadInfo()
		if err != nil {
			return err
		}

		var challenge []byte
		if encoded != "" {
			challenge, err = base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return err
			}
		}

		res, err := sasl.Next(challenge)
		if err != nil {
			return err
		}

		if res == nil {
			return nil
		}

		encoded = base64.StdEncoding.EncodeToString(res)
		cont := &imap.ContinuationResp{Info: encoded}
		if err := cont.WriteTo(w); err != nil {
			return err
		}
	}
}
