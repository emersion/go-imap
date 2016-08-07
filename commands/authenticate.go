package commands

import (
	"encoding/base64"
	"errors"
	"strings"

	imap "github.com/emersion/go-imap/common"
	"github.com/emersion/go-sasl"
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

func (cmd *Authenticate) Handle(mechanisms map[string]sasl.Server, r *imap.Reader, w imap.Writer) (err error) {
	sasl, ok := mechanisms[cmd.Mechanism]
	if !ok {
		err = errors.New("Unsupported mechanism")
		return
	}

	var response []byte
	for {
		var challenge []byte
		var done bool
		challenge, done, err = sasl.Next(response)
		if err != nil || done {
			return
		}

		encoded := base64.StdEncoding.EncodeToString(challenge)
		cont := &imap.ContinuationResp{Info: encoded}
		if err = cont.WriteTo(w); err != nil {
			return
		}

		if encoded, err = r.ReadInfo(); err != nil {
			return
		}
		if encoded != "" {
			response, err = base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return
			}
		}
	}
}
