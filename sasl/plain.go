package sasl

import (
	"bytes"
	"errors"

	"github.com/emersion/imap/backend"
)

type plainClient struct {
	Username string
	Password string
	Identity string
}

func (a *plainClient) Start() (mech string, ir []byte, err error) {
	mech = "PLAIN"
	ir = []byte(a.Identity + "\x00" + a.Username + "\x00" + a.Password)
	return
}

func (a *plainClient) Next(challenge []byte) (response []byte, err error) {
	return nil, errors.New("unexpected server challenge")
}

func NewPlainClient(username, password, identity string) Client {
	return &plainClient{username, password, identity}
}

type plainServer struct {
	done bool
	user backend.User
	backend backend.Backend
}

func (a *plainServer) Start() (ir []byte, err error) {
	return []byte{}, nil
}

func (a *plainServer) Next(challenge []byte) (response []byte, err error) {
	if a.done {
		err = errors.New("unexpected client challenge")
		return
	}

	a.done = true

	parts := bytes.Split(challenge, []byte("\x00"))
	if len(parts) != 3 {
		err = errors.New("invalid challenge")
		return
	}

	// TODO: support identity
	identity := string(parts[0])
	if identity != "" {
		err = errors.New("SASL identity is not supported")
		return
	}

	username := string(parts[1])
	password := string(parts[2])

	user, err := a.backend.Login(username, password)
	if err != nil {
		return
	}

	a.user = user
	return
}

func (a *plainServer) User() backend.User {
	return a.user
}

func NewPlainServer(bkd backend.Backend) Server {
	return &plainServer{backend: bkd}
}
