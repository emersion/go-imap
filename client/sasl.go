package client

import (
	"errors"

	imap "github.com/emersion/imap/common"
)

// A common.SaslClient implementing PLAIN authentication.
type plainSasl struct {
	Username string
	Password string
	Identity string
}

func (a *plainSasl) Start() (mech string, ir []byte, err error) {
	mech = "PLAIN"
	ir = []byte(a.Identity + "\x00" + a.Username + "\x00" + a.Password)
	return
}

func (a *plainSasl) Next(challenge []byte) (response []byte, err error) {
	return nil, errors.New("unexpected server challenge")
}

func NewPlainSasl(username, password, identity string) imap.SaslClient {
	return &plainSasl{username, password, identity}
}
