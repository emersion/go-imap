package commands

import (
	"bufio"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-sasl"
)

// AuthenticateConn is a connection that supports IMAP authentication.
type AuthenticateConn interface {
	io.Reader

	// WriteResp writes an IMAP response to this connection.
	WriteResp(res imap.WriterTo) error
}

// Authenticate is an AUTHENTICATE command, as defined in RFC 3501 section
// 6.2.2.
type Authenticate struct {
	Mechanism       string
	InitialResponse string
}

func (cmd *Authenticate) Command() *imap.Command {
	if cmd.InitialResponse != "" {
		return &imap.Command{
			Name:      "AUTHENTICATE",
			Arguments: []interface{}{imap.RawString(cmd.Mechanism), imap.RawString(cmd.InitialResponse)},
		}
	}
	return &imap.Command{
		Name:      "AUTHENTICATE",
		Arguments: []interface{}{imap.RawString(cmd.Mechanism)},
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

	if len(fields) != 2 {
		return nil
	}

	if cmd.InitialResponse, ok = fields[1].(string); !ok {
		return errors.New("InitialResponse must be a string")
	}

	return nil
}

func (cmd *Authenticate) Handle(mechanisms map[string]sasl.Server, conn AuthenticateConn) error {
	sasl, ok := mechanisms[cmd.Mechanism]
	if !ok {
		return errors.New("Unsupported mechanism")
	}

	scanner := bufio.NewScanner(conn)

	var response []byte
	var err error
	if cmd.InitialResponse != "" {
		// "=" is invalid base64 but it is used to indicate empty initial
		// response.
		if cmd.InitialResponse == "=" {
			response = []byte{}
		} else {
			response, err = base64.StdEncoding.DecodeString(cmd.InitialResponse)
			if err != nil {
				return err
			}
		}
	}
	for {
		challenge, done, err := sasl.Next(response)
		if err != nil || done {
			return err
		}

		encoded := base64.StdEncoding.EncodeToString(challenge)
		cont := &imap.ContinuationReq{Info: encoded}
		if err := conn.WriteResp(cont); err != nil {
			return err
		}

		scanner.Scan()
		if err := scanner.Err(); err != nil {
			return err
		}

		encoded = scanner.Text()
		if encoded != "" {
			response, err = base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return err
			}
		}
	}
}
