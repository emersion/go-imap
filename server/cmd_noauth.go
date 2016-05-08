package server

import (
	"errors"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/sasl"
)

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

	Mechanisms map[string]sasl.Server
}

func (cmd *Authenticate) Handle(conn *Conn) error {
	if conn.State != common.NotAuthenticatedState {
		return errors.New("Already authenticated")
	}
	if !conn.CanAuth() {
		return errors.New("Authentication disabled")
	}

	user, err := cmd.Authenticate.Handle(cmd.Mechanisms, conn.Reader, conn.Writer)
	if err != nil {
		return err
	}

	conn.State = common.AuthenticatedState
	conn.User = user
	return nil
}
