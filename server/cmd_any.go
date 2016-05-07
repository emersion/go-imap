package server

import (
	"github.com/emersion/imap/commands"
	"github.com/emersion/imap/responses"
)

type Capability struct {
	commands.Capability
}

func (cmd *Capability) Handle(conn *Conn) error {
	res := &responses.Capability{
		Caps: conn.getCaps(),
	}

	return res.Response().WriteTo(conn.Writer)
}

type Noop struct {
	commands.Noop
}

func (cmd *Noop) Handle(conn *Conn) error {
	return nil
}
