package imapserver

import (
	"fmt"

	"github.com/emersion/go-sasl"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *conn) handleAuthenticate(dec *imapwire.Decoder) error {
	var mech string
	if !dec.ExpectSP() || !dec.ExpectAtom(&mech) {
		return dec.Err()
	}

	var initialResp []byte
	if dec.SP() {
		var initialRespStr string
		if !dec.ExpectText(&initialRespStr) {
			return dec.Err()
		}
		var err error
		initialResp, err = internal.DecodeSASL(initialRespStr)
		if err != nil {
			return err
		}
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateNotAuthenticated); err != nil {
		return err
	}

	// TODO: support other SASL mechanisms
	if mech != "PLAIN" {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "SASL mechanism not supported",
		}
	}

	enc := newResponseEncoder(c)
	defer enc.end()

	saslServer := sasl.NewPlainServer(func(identity, username, password string) error {
		if identity != "" && identity != username {
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeAuthorizationFailed,
				Text: "SASL identity not supported",
			}
		}
		return c.session.Login(username, password)
	})

	resp := initialResp
	for {
		challenge, done, err := saslServer.Next(resp)
		if err != nil {
			return err
		} else if done {
			break
		}

		var challengeStr string
		if challenge != nil {
			challengeStr = internal.EncodeSASL(challenge)
		}
		if err := writeContReq(enc.Encoder, challengeStr); err != nil {
			return err
		}

		respStr, isPrefix, err := c.br.ReadLine()
		if err != nil {
			return err
		} else if isPrefix {
			return fmt.Errorf("SASL response too long")
		} else if string(respStr) == "*" {
			return &imap.Error{
				Type: imap.StatusResponseTypeBad,
				Text: "AUTHENTICATE cancelled",
			}
		}

		resp, err = decodeSASL(string(respStr))
		if err != nil {
			return err
		}
	}

	c.state = imap.ConnStateAuthenticated
	return nil
}

func decodeSASL(s string) ([]byte, error) {
	b, err := internal.DecodeSASL(s)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Malformed SASL response",
		}
	}
	return b, nil
}
