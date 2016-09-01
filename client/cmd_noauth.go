package client

import (
	"crypto/tls"
	"errors"
	"net"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
	"github.com/emersion/go-sasl"
)

var (
	ErrAlreadyLoggedIn   = errors.New("Already logged in")
	ErrTLSAlreadyEnabled = errors.New("TLS is already enabled")
	ErrLoginDisabled     = errors.New("Login is disabled in current state")
)

// Check if the server supports STARTTLS.
func (c *Client) SupportsStartTLS() bool {
	return c.Caps[imap.StartTLS]
}

// If the connection to the IMAP server isn't secure, starts TLS negotiation.
//
// This function also resets c.Caps because capabilities change when TLS is
// enabled.
func (c *Client) StartTLS(tlsConfig *tls.Config) (err error) {
	if c.isTLS {
		err = ErrTLSAlreadyEnabled
		return
	}

	cmd := &commands.StartTLS{}

	err = c.Upgrade(func(conn net.Conn) (net.Conn, error) {
		if status, err := c.execute(cmd, nil); err != nil {
			return nil, err
		} else if err := status.Err(); err != nil {
			return nil, err
		}

		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			return nil, err
		}

		c.Caps = nil
		return tlsConn, nil
	})
	if err != nil {
		return
	}

	c.isTLS = true
	return
}

// Check if the server supports a given authentication mechanism.
func (c *Client) SupportsAuth(mech string) bool {
	return c.Caps["AUTH="+mech]
}

// Indicates a SASL authentication mechanism to the server. If the server
// supports the requested authentication mechanism, it performs an
// authentication protocol exchange to authenticate and identify the client.
func (c *Client) Authenticate(auth sasl.Client) (err error) {
	if c.State != imap.NotAuthenticatedState {
		err = ErrAlreadyLoggedIn
		return
	}

	mech, ir, err := auth.Start()
	if err != nil {
		return
	}

	cmd := &commands.Authenticate{
		Mechanism: mech,
	}

	res := &responses.Authenticate{
		Mechanism:       auth,
		InitialResponse: ir,
		Writer:          c.conn.Writer,
	}

	status, err := c.execute(cmd, res)
	if err != nil {
		return
	}
	if err = status.Err(); err != nil {
		return
	}

	c.State = imap.AuthenticatedState
	c.Caps = nil

	if status.Code == "CAPABILITY" {
		c.gotStatusCaps(status.Arguments)
	}

	return
}

// Identifies the client to the server and carries the plaintext password
// authenticating this user.
func (c *Client) Login(username, password string) (err error) {
	if c.State != imap.NotAuthenticatedState {
		err = ErrAlreadyLoggedIn
		return
	}
	if c.Caps["LOGINDISABLED"] {
		err = ErrLoginDisabled
		return
	}

	cmd := &commands.Login{
		Username: username,
		Password: password,
	}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}
	if err = status.Err(); err != nil {
		return
	}

	c.State = imap.AuthenticatedState
	c.Caps = nil

	if status.Code == "CAPABILITY" {
		c.gotStatusCaps(status.Arguments)
	}

	return
}
