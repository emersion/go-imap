package client

import (
	"errors"
)

// A common.SaslClient implementing PLAIN authentication.
type PlainSasl struct {
	Username string
	Password string
	Identity string
}

func (a *PlainSasl) Start() (mech string, ir []byte, err error) {
	mech = "PLAIN"
	ir = []byte(a.Identity + "\x00" + a.Username + "\x00" + a.Password)
	return
}

func (a *PlainSasl) Next(challenge []byte) (response []byte, err error) {
	return nil, errors.New("unexpected server challenge")
}

func NewPlainSasl(username, password, identity string) *PlainSasl {
	return &PlainSasl{username, password, identity}
}
