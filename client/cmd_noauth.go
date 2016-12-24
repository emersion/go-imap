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
	// ErrAlreadyLoggedIn is returned if Login or Authenticate is called when the
	// client is already logged in.
	ErrAlreadyLoggedIn = errors.New("Already logged in")
	// ErrTLSAlreadyEnabled is returned if StartTLS is called when TLS is already
	// enabled.
	ErrTLSAlreadyEnabled = errors.New("TLS is already enabled")
	// ErrLoginDisabled is returned if Login or Authenticate is called when the
	// server has disabled authentication. Most of the time, calling enabling TLS
	// solves the problem.
	ErrLoginDisabled = errors.New("Login is disabled in current state")
)

// SupportsStartTLS checks if the server supports STARTTLS.
func (c *Client) SupportsStartTLS() bool {
	c.capsLocker.Lock()
	tls := c.Caps[imap.StartTLS]
	c.capsLocker.Unlock()
	return tls
}

// StartTLS starts TLS negotiation.
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

		c.capsLocker.Lock()
		c.Caps = nil
		c.capsLocker.Unlock()

		return tlsConn, nil
	})
	if err != nil {
		return
	}

	c.isTLS = true
	return
}

// SupportsAuth checks if the server supports a given authentication mechanism.
func (c *Client) SupportsAuth(mech string) bool {
	c.capsLocker.Lock()
	auth := c.Caps["AUTH="+mech]
	c.capsLocker.Unlock()
	return auth
}

// Authenticate indicates a SASL authentication mechanism to the server. If the
// server supports the requested authentication mechanism, it performs an
// authentication protocol exchange to authenticate and identify the client.
func (c *Client) Authenticate(auth sasl.Client) error {
	if c.State != imap.NotAuthenticatedState {
		return ErrAlreadyLoggedIn
	}

	mech, ir, err := auth.Start()
	if err != nil {
		return err
	}

	cmd := &commands.Authenticate{
		Mechanism: mech,
	}

	res := &responses.Authenticate{
		Mechanism:       auth,
		InitialResponse: ir,
		Writer:          c.Writer(),
	}

	status, err := c.execute(cmd, res)
	if err != nil {
		return err
	}
	if err = status.Err(); err != nil {
		return err
	}

	c.State = imap.AuthenticatedState

	c.capsLocker.Lock()
	c.Caps = nil
	c.capsLocker.Unlock()

	if status.Code == "CAPABILITY" {
		c.gotStatusCaps(status.Arguments)
	}

	return nil
}

// Login identifies the client to the server and carries the plaintext password
// authenticating this user.
func (c *Client) Login(username, password string) (err error) {
	if c.State != imap.NotAuthenticatedState {
		err = ErrAlreadyLoggedIn
		return
	}

	c.capsLocker.Lock()
	if c.Caps["LOGINDISABLED"] {
		c.capsLocker.Unlock()
		err = ErrLoginDisabled
		return
	}
	c.capsLocker.Unlock()

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

	c.capsLocker.Lock()
	c.Caps = nil
	c.capsLocker.Unlock()

	if status.Code == "CAPABILITY" {
		c.gotStatusCaps(status.Arguments)
	}

	return
}
