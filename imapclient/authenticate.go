package imapclient

import (
	"encoding/base64"
	"fmt"

	"github.com/emersion/go-sasl"
)

// Authenticate sends an AUTHENTICATE command.
//
// Unlike other commands, this method blocks until the SASL exchange completes.
func (c *Client) Authenticate(saslClient sasl.Client) error {
	mech, initialResp, err := saslClient.Start()
	if err != nil {
		return err
	}

	cmd := &Command{}
	contReq := c.registerContReq(cmd)
	enc := c.beginCommand("AUTHENTICATE", cmd)
	enc.SP().Atom(mech)
	if initialResp != nil {
		enc.SP().Atom(encodeSASL(initialResp))
	}
	enc.flush()
	defer enc.end()

	for {
		challengeStr, err := contReq.Wait()
		if err != nil {
			return cmd.Wait()
		}

		if challengeStr == "" {
			if initialResp == nil {
				return fmt.Errorf("imapclient: server requested SASL initial response, but we don't have one")
			}

			contReq = c.registerContReq(cmd)
			if err := c.writeSASLResp(initialResp); err != nil {
				return err
			}
			initialResp = nil
			continue
		}

		challenge, err := decodeSASL(challengeStr)
		if err != nil {
			return err
		}

		resp, err := saslClient.Next(challenge)
		if err != nil {
			return err
		}

		contReq = c.registerContReq(cmd)
		if err := c.writeSASLResp(resp); err != nil {
			return err
		}
	}
}

func (c *Client) writeSASLResp(resp []byte) error {
	respStr := encodeSASL(resp)
	if _, err := c.bw.WriteString(respStr + "\r\n"); err != nil {
		return err
	}
	if err := c.bw.Flush(); err != nil {
		return err
	}
	return nil
}

func encodeSASL(b []byte) string {
	if len(b) == 0 {
		return "="
	} else {
		return base64.StdEncoding.EncodeToString(b)
	}
}

func decodeSASL(s string) ([]byte, error) {
	if s == "=" {
		return []byte{}, nil
	} else {
		return base64.StdEncoding.DecodeString(s)
	}
}
