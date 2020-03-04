package server

import (
	"compress/flate"
	"net"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/commands"
	"github.com/emersion/go-imap/internal"
	"github.com/emersion/go-imap/responses"
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

type Compress struct {
	commands.Compress
}

func (cmd *Compress) Handle(conn Conn) error {
	if cmd.Mechanism != imap.CompressDeflate {
		return imap.CompressUnsupportedError{Mechanism: cmd.Mechanism}
	}
	return nil
}

func (cmd *Compress) Upgrade(conn Conn) error {
	err := conn.Upgrade(func(conn net.Conn) (net.Conn, error) {
		return internal.CreateDeflateConn(conn, flate.DefaultCompression)
	})
	if err != nil {
		return err
	}

	return nil
}
