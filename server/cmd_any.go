package server

import (
	"github.com/faziloub/go-imap"
	"github.com/faziloub/go-imap/backend"
	"github.com/faziloub/go-imap/commands"
	"github.com/faziloub/go-imap/responses"
)

type Capability struct {
	commands.Capability
}

func (cmd *Capability) Handle(conn Conn) error {
	res := &responses.Capability{Caps: conn.Capabilities()}
	return conn.WriteResp(res)
}

type Noop struct {
	commands.Noop
}

func (cmd *Noop) Handle(conn Conn) error {
	ctx := conn.Context()
	if ctx.Mailbox != nil {
		// If a mailbox is selected, NOOP can be used to poll for server updates
		if mbox, ok := ctx.Mailbox.(backend.MailboxPoller); ok {
			return mbox.Poll()
		}
	}

	return nil
}

type Logout struct {
	commands.Logout
}

func (cmd *Logout) Handle(conn Conn) error {
	res := &imap.StatusResp{
		Type: imap.StatusRespBye,
		Info: "Closing connection",
	}

	if err := conn.WriteResp(res); err != nil {
		return err
	}

	// Request to close the connection
	conn.Context().State = imap.LogoutState
	return nil
}
