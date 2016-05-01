package client

import (
	"crypto/tls"
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
)

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
	return
}

// TODO: AUTHENTICATE

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

	if status.Code == "CAPABILITY" {
		c.gotStatusCaps(status.Arguments)
	}

	c.State = imap.AuthenticatedState

	return
}
