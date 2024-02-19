package imapserver

import (
	"fmt"
	"strings"

	"github.com/emersion/go-sasl"

	"github.com/opsxolc/go-imap/v2"
	"github.com/opsxolc/go-imap/v2/internal"
	"github.com/opsxolc/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleAuthenticate(tag string, dec *imapwire.Decoder) error {
	var mech string
	if !dec.ExpectSP() || !dec.ExpectAtom(&mech) {
		return dec.Err()
	}
	mech = strings.ToUpper(mech)

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
	if !c.canAuth() {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodePrivacyRequired,
			Text: "TLS is required to authenticate",
		}
	}

	var saslServer sasl.Server
	if authSess, ok := c.session.(SessionSASL); ok {
		var err error
		saslServer, err = authSess.Authenticate(mech)
		if err != nil {
			return err
		}
	} else {
		if mech != "PLAIN" {
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Text: "SASL mechanism not supported",
			}
		}
		saslServer = sasl.NewPlainServer(func(identity, username, password string) error {
			if identity != "" && identity != username {
				return &imap.Error{
					Type: imap.StatusResponseTypeNo,
					Code: imap.ResponseCodeAuthorizationFailed,
					Text: "SASL identity not supported",
				}
			}
			return c.session.Login(username, password)
		})
	}

	enc := newResponseEncoder(c)
	defer enc.end()

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

		encodedResp, isPrefix, err := c.br.ReadLine()
		if err != nil {
			return err
		} else if isPrefix {
			return fmt.Errorf("SASL response too long")
		} else if string(encodedResp) == "*" {
			return &imap.Error{
				Type: imap.StatusResponseTypeBad,
				Text: "AUTHENTICATE cancelled",
			}
		}

		resp, err = decodeSASL(string(encodedResp))
		if err != nil {
			return err
		}
	}

	c.state = imap.ConnStateAuthenticated
	text := fmt.Sprintf("%v authentication successful", mech)
	return writeCapabilityOK(enc.Encoder, tag, c.availableCaps(), text)
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

func (c *Conn) handleUnauthenticate(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}
	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}
	session, ok := c.session.(SessionUnauthenticate)
	if !ok {
		return newClientBugError("UNAUTHENTICATE is not supported")
	}
	if err := session.Unauthenticate(); err != nil {
		return err
	}
	c.state = imap.ConnStateNotAuthenticated
	c.mutex.Lock()
	c.enabled = make(imap.CapSet)
	c.mutex.Unlock()
	return nil
}
