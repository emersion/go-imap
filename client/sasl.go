package client

import (
	"errors"
)

// A common.SaslClient implementing PLAIN authentication.
type SaslPlain struct {
	Username string
	Password string
	Identity string
}

func (a *SaslPlain) Start() (mech string, ir []byte, err error) {
	mech = "PLAIN"
	ir = []byte(a.Identity + "\x00" + a.Username + "\x00" + a.Password)
	return
}

func (a *SaslPlain) Next(challenge []byte) (response []byte, err error) {
	return nil, errors.New("unexpected server challenge")
}
