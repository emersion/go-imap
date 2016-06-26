package server

import (
	"crypto/tls"
	"errors"
	"net"

	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-sasl"
)

// Common errors in Not Authenticated state.
var (
	ErrAlreadyAuthenticated = errors.New("Already authenticated")
	ErrAuthDisabled = errors.New("Authentication disabled")
)

type StartTLS struct {
	commands.StartTLS
}

func (cmd *StartTLS) Handle(conn *Conn) error {
	if conn.State != common.NotAuthenticatedState {
		return ErrAlreadyAuthenticated
	}
	if conn.IsTLS() {
		return errors.New("TLS is already enabled")
	}
	if conn.Server.TLSConfig == nil {
		return errors.New("TLS support not enabled")
	}

	// First send an OK status response to let the client know that the TLS
	// Handshake can begin
	// TODO: set the Tag!
	res := &common.StatusResp{
		Type: common.OK,
		Info: "Begin TLS negotiation now",
	}
	if err := conn.WriteRes(res); err != nil {
		return err
	}

	tlsConfig := conn.Server.TLSConfig
	upgrader := func (conn net.Conn) (net.Conn, error) {
		upgraded := tls.Server(conn, tlsConfig)
		err := upgraded.Handshake()
		return upgraded, err
	}

	if err := conn.Upgrade(upgrader); err != nil {
		return err
	}

	conn.isTLS = true
	return ErrNoStatusResp()
}

func afterAuthStatus(conn *Conn) error {
	return ErrStatusResp(&common.StatusResp{
		Type: common.OK,
		Code: common.Capability,
		Arguments: common.FormatStringList(conn.getCaps()),
	})
}

type Login struct {
	commands.Login
}

func (cmd *Login) Handle(conn *Conn) error {
	if conn.State != common.NotAuthenticatedState {
		return ErrAlreadyAuthenticated
	}
	if !conn.CanAuth() {
		return ErrAuthDisabled
	}

	user, err := conn.Server.Backend.Login(cmd.Username, cmd.Password)
	if err != nil {
		return err
	}

	conn.State = common.AuthenticatedState
	conn.User = user
	return afterAuthStatus(conn)
}

type Authenticate struct {
	commands.Authenticate
}

func (cmd *Authenticate) Handle(conn *Conn) error {
	if conn.State != common.NotAuthenticatedState {
		return ErrAlreadyAuthenticated
	}
	if !conn.CanAuth() {
		return ErrAuthDisabled
	}

	mechanisms := map[string]sasl.Server{}
	for name, newSasl := range conn.Server.auths {
		mechanisms[name] = newSasl(conn)
	}

	err := cmd.Authenticate.Handle(mechanisms, conn.Reader, conn.Writer)
	if err != nil {
		return err
	}

	return afterAuthStatus(conn)
}
