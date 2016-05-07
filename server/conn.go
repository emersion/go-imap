package server

import (
	"bufio"
	"net"

	imap "github.com/emersion/imap/common"
)

type Conn struct {
	*imap.Reader
	*imap.Writer

	conn net.Conn

	Server *Server
}

func newConn(s *Server, c net.Conn) *Conn {
	r := imap.NewReader(bufio.NewReader(c))
	w := imap.NewWriter(c)

	return &Conn{
		Reader: r,
		Writer: w,

		conn: c,

		Server: s,
	}
}
