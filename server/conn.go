package server

import (
	"bufio"
	"crypto/tls"
	"net"

	imap "github.com/emersion/imap/common"
)

type Conn struct {
	*imap.Reader
	*imap.Writer

	conn net.Conn

	Server *Server
	State imap.ConnState
}

func (c *Conn) getCaps() (caps []string) {
	caps = []string{"IMAP4rev1"}

	if !c.IsTLS() {
		caps = append(caps, imap.StartTLS, "LOGINDISABLED")
		return
	}

	caps = append(caps, "AUTH=PLAIN")
	return
}

func (c *Conn) greet() error {
	caps := c.getCaps()
	args := make([]interface{}, len(caps))
	for i, cap := range caps {
		args[i] = cap
	}

	greeting := &imap.StatusResp{
		Tag: "*",
		Type: imap.OK,
		Code: imap.Capability,
		Arguments: args,
		Info: "IMAP4rev1 Service Ready",
	}

	return greeting.WriteTo(c.Writer)
}

func (c *Conn) IsTLS() bool {
	_, ok := c.conn.(*tls.Conn)
	return ok
}

func newConn(s *Server, c net.Conn) *Conn {
	r := imap.NewReader(bufio.NewReader(c))
	w := imap.NewWriter(c)

	return &Conn{
		Reader: r,
		Writer: w,

		conn: c,

		Server: s,
		State: imap.NotAuthenticatedState,
	}
}
