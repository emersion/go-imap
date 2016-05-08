package server

import (
	"errors"

	imap "github.com/emersion/imap/common"
	"github.com/emersion/imap/commands"
)

type Login struct {
	commands.Login
}

func (cmd *Login) Handle(conn *Conn, bkd Backend) error {
	if conn.State != imap.NotAuthenticatedState {
		return errors.New("Already authenticated")
	}

	if err := bkd.Login(cmd.Username, cmd.Password); err != nil {
		return err
	}

	conn.State = imap.AuthenticatedState
	return nil
}
