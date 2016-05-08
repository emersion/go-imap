package server

import (
	"bytes"
	"errors"
)

type PlainSasl struct {
	done bool
	backend Backend
}

func (a *PlainSasl) Start() (ir []byte, err error) {
	return []byte{}, nil
}

func (a *PlainSasl) Next(challenge []byte) (response []byte, err error) {
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

	err = a.backend.Login(username, password)
	return
}
