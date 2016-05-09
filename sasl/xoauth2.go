package sasl

import (
	"errors"
)

type xoauth2Client struct {
	Username string
	Token string
}

func (a *xoauth2Client) Start() (mech string, ir []byte, err error) {
	mech = "XOAUTH2"
	ir = []byte("user=" + a.Username + "\x01auth=Bearer " + a.Token + "\x01\x01")
	return
}

func (a *xoauth2Client) Next(challenge []byte) (response []byte, err error) {
	return nil, errors.New("unexpected server challenge")
}

// An implementation of the XOAUTH2 authentication mechanism, as
// described in https://developers.google.com/gmail/xoauth2_protocol.
func NewXoauth2Client(username, token string) Client {
	return &xoauth2Client{username, token}
}
