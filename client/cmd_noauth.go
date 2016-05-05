package client

import (
	"crypto/tls"
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
)

// If the connection to the IMAP server isn't secure, starts TLS negociation.
func (c *Client) StartTLS(tlsConfig *tls.Config) (err error) {
	if _, ok := c.conn.(*tls.Conn); ok {
		err = errors.New("TLS is already enabled")
		return
	}

	cmd := &commands.StartTLS{}

	status, err := c.execute(cmd, nil)
	if err != nil {
		return
	}
	if err = status.Err(); err != nil {
		return
	}

	tlsConn := tls.Client(c.conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		return
	}

	c.conn = tlsConn
	c.Caps = nil
	return
}

// Indicates a SASL authentication mechanism to the server. If the server
// supports the requested authentication mechanism, it performs an
// authentication protocol exchange to authenticate and identify the client.
func (c *Client) Authenticate(auth imap.Sasl) (err error) {
	if c.State != imap.NotAuthenticatedState {
		err = errors.New("Already logged in")
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
		Mechanism: auth,
		InitialResponse: ir,
		Writer: c.writer,
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
		err = errors.New("Already logged in")
		return
	}
	if c.Caps["LOGINDISABLED"] {
		err = errors.New("Login is disabled in current state")
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
