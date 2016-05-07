package server

import (
	"github.com/emersion/imap/commands"
)

type Noop struct {
	commands.Noop
}

func (cmd *Noop) Handle(conn *Conn) error {
	return nil
}
