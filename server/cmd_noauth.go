package server

import (
	"crypto/tls"
	"errors"

	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-sasl"
)

type StartTLS struct {
	commands.StartTLS
}

func (cmd *StartTLS) Handle(conn *Conn) error {
	if conn.State != common.NotAuthenticatedState {
		return errors.New("Already authenticated")
	}
	if conn.IsTLS() {
		return errors.New("TLS is already enabled")
	}
	if conn.Server.TLSConfig == nil {
		return errors.New("TLS support not enabled")
	}

	upgraded := tls.Server(conn.conn, conn.Server.TLSConfig)

	if err := upgraded.Handshake(); err != nil {
		return err
	}

	conn.conn = upgraded
	return nil
}

type Login struct {
	commands.Login
}

func (cmd *Login) Handle(conn *Conn) error {
	if conn.State != common.NotAuthenticatedState {
		return errors.New("Already authenticated")
	}
	if !conn.CanAuth() {
		return errors.New("Authentication disabled")
	}

	user, err := conn.Server.Backend.Login(cmd.Username, cmd.Password)
	if err != nil {
		return err
	}

	conn.State = common.AuthenticatedState
	conn.User = user
	return nil
}

type Authenticate struct {
	commands.Authenticate
}

func (cmd *Authenticate) Handle(conn *Conn) error {
	if conn.State != common.NotAuthenticatedState {
		return errors.New("Already authenticated")
	}
	if !conn.CanAuth() {
		return errors.New("Authentication disabled")
	}

	mechanisms := map[string]sasl.Server{}
	for name, newSasl := range conn.Server.auths {
		mechanisms[name] = newSasl(conn)
	}

	return cmd.Authenticate.Handle(mechanisms, conn.Reader, conn.Writer)
}
