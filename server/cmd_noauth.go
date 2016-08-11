package server

import (
	"crypto/tls"
	"errors"
	"net"

	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
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

func (cmd *StartTLS) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.State != common.NotAuthenticatedState {
		return ErrAlreadyAuthenticated
	}
	if conn.IsTLS() {
		return errors.New("TLS is already enabled")
	}
	if conn.Server().TLSConfig == nil {
		return errors.New("TLS support not enabled")
	}

	// Send an OK status response to let the client know that the TLS handshake
	// can begin
	return ErrStatusResp(&common.StatusResp{
		Type: common.StatusOk,
		Info: "Begin TLS negotiation now",
	})
}

func (cmd *StartTLS) Upgrade(conn Conn) error {
	tlsConfig := conn.Server().TLSConfig

	var tlsConn *tls.Conn
	err := conn.Upgrade(func (conn net.Conn) (net.Conn, error) {
		tlsConn = tls.Server(conn, tlsConfig)
		err := tlsConn.Handshake()
		return tlsConn, err
	})
	if err != nil {
		return err
	}

	conn.conn().tlsConn = tlsConn

	res := &responses.Capability{Caps: conn.Capabilities()}
	return conn.WriteResp(res)
}

func afterAuthStatus(conn Conn) error {
	return ErrStatusResp(&common.StatusResp{
		Type: common.StatusOk,
		Code: common.CodeCapability,
		Arguments: common.FormatStringList(conn.Capabilities()),
	})
}

func canAuth(conn Conn) bool {
	for _, cap := range conn.Capabilities() {
		if cap == "AUTH=PLAIN" {
			return true
		}
	}
	return false
}

type Login struct {
	commands.Login
}

func (cmd *Login) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.State != common.NotAuthenticatedState {
		return ErrAlreadyAuthenticated
	}
	if !canAuth(conn) {
		return ErrAuthDisabled
	}

	user, err := conn.Server().Backend.Login(cmd.Username, cmd.Password)
	if err != nil {
		return err
	}

	ctx.State = common.AuthenticatedState
	ctx.User = user
	return afterAuthStatus(conn)
}

type Authenticate struct {
	commands.Authenticate
}

func (cmd *Authenticate) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.State != common.NotAuthenticatedState {
		return ErrAlreadyAuthenticated
	}
	if !canAuth(conn) {
		return ErrAuthDisabled
	}

	mechanisms := map[string]sasl.Server{}
	for name, newSasl := range conn.Server().auths {
		mechanisms[name] = newSasl(conn)
	}

	// TODO: do not use Reader and Writer here
	err := cmd.Authenticate.Handle(mechanisms, conn.conn().Reader, conn.conn().Writer)
	if err != nil {
		return err
	}

	return afterAuthStatus(conn)
}
