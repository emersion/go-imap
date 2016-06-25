package server

import (
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/responses"
)

type Capability struct {
	commands.Capability
}

func (cmd *Capability) Handle(conn *Conn) error {
	res := &responses.Capability{
		Caps: conn.getCaps(),
	}

	return conn.WriteRes(res.Response())
}

type Noop struct {
	commands.Noop
}

func (cmd *Noop) Handle(conn *Conn) error {
	if conn.Mailbox != nil {
		// If a mailbox is selected, NOOP can be used to poll for server updates
		if mbox, ok := conn.Mailbox.(backend.UpdaterMailbox); ok {
			return mbox.Poll()
		}
	}

	return nil
}

type Logout struct {
	commands.Logout
}

func (cmd *Logout) Handle(conn *Conn) error {
	res := &common.StatusResp{
		Tag: "*",
		Type: common.BYE,
		Info: "Closing connection",
	}

	if err := conn.WriteRes(res); err != nil {
		return err
	}

	// Request to close the connection
	conn.State = common.LogoutState
	return nil
}
