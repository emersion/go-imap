package imapclient

import (
	"encoding/base64"
	"fmt"

	"github.com/emersion/go-sasl"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
)

// Authenticate sends an AUTHENTICATE command.
//
// Unlike other commands, this method blocks until the SASL exchange completes.
func (c *Client) Authenticate(saslClient sasl.Client) error {
	mech, initialResp, err := saslClient.Start()
	if err != nil {
		return err
	}

	// c.Caps may send a CAPABILITY command, so check it before c.beginCommand
	var hasSASLIR bool
	if initialResp != nil {
		hasSASLIR = c.Caps().Has(imap.CapSASLIR)
	}

	cmd := &authenticateCommand{}
	contReq := c.registerContReq(cmd)
	enc := c.beginCommand("AUTHENTICATE", cmd)
	enc.SP().Atom(mech)
	if initialResp != nil && hasSASLIR {
		enc.SP().Atom(internal.EncodeSASL(initialResp))
		initialResp = nil
	}
	enc.flush()
	defer enc.end()

	for {
		challengeStr, err := contReq.Wait()
		if err != nil {
			return cmd.wait()
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

		challenge, err := internal.DecodeSASL(challengeStr)
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

type authenticateCommand struct {
	commandBase
}

func (c *Client) writeSASLResp(resp []byte) error {
	// A continuation response is base64 (RFC 3501). A zero-length response must be
	// an empty line: the "=" zero-length marker (RFC 4959) is valid only in the
	// AUTHENTICATE initial-response slot, not in a continuation, where strict
	// servers (e.g. Dovecot) reject "=" as invalid base64. This matters for
	// multi-step mechanisms like GSSAPI, whose mutual-auth step sends an empty token.
	respStr := base64.StdEncoding.EncodeToString(resp)
	if _, err := c.bw.WriteString(respStr + "\r\n"); err != nil {
		return err
	}
	if err := c.bw.Flush(); err != nil {
		return err
	}
	return nil
}

// Unauthenticate sends an UNAUTHENTICATE command.
//
// This command requires support for the UNAUTHENTICATE extension.
func (c *Client) Unauthenticate() *Command {
	cmd := &unauthenticateCommand{}
	c.beginCommand("UNAUTHENTICATE", cmd).end()
	return &cmd.Command
}

type unauthenticateCommand struct {
	Command
}
