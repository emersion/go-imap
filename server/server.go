// An IMAP server.
package server

import (
	"errors"
	"io"
	"log"
	"net"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/backend"
	"github.com/emersion/imap/sasl"
)

// A command handler.
type Handler interface {
	common.Parser

	// Handle this command for a given connection.
	Handle(conn *Conn) error
}

// A function that creates handlers.
type HandlerFactory func () Handler

// An IMAP server.
type Server struct {
	listener net.Listener
	commands map[string]HandlerFactory
	auths map[string]sasl.Server

	// This server's backend.
	Backend backend.Backend
	// Allow authentication over unencrypted connections.
	AllowInsecureAuth bool
}

// Get this server's address.
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) listen() error {
	defer s.listener.Close()

	for {
		c, err := s.listener.Accept()
		if err != nil {
			return err
		}

		log.Println("New conn", c.RemoteAddr())

		conn := newConn(s, c)

		go (func () {
			s.handleConn(conn)
			log.Println("Connection closed")
		})()
	}
}

func (s *Server) handleConn(conn *Conn) error {
	// Send greeting
	if err := conn.greet(); err != nil {
		return err
	}

	for {
		if conn.State == common.LogoutState {
			return conn.Close()
		}

		fields, err := conn.ReadLine()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Println("Error reading command:", err)
			continue
		}

		var res common.WriterTo

		cmd := &common.Command{}
		if err := cmd.Parse(fields); err != nil {
			res = &common.StatusResp{
				Tag: "*",
				Type: common.BAD,
				Info: err.Error(),
			}
		} else {
			res, err = s.handleCommand(cmd, conn)
			if err != nil {
				res = &common.StatusResp{
					Tag: cmd.Tag,
					Type: common.BAD,
					Info: err.Error(),
				}
			}
		}

		if err := res.WriteTo(conn.Writer); err != nil {
			log.Println("Error writing response:", err)
			continue
		}
	}
}

func (s *Server) handleCommand(cmd *common.Command, conn *Conn) (res common.WriterTo, err error) {
	newHandler, ok := s.commands[cmd.Name]
	if !ok {
		err = errors.New("Unknown command")
		return
	}

	handler := newHandler()
	if err = handler.Parse(cmd.Arguments); err != nil {
		return
	}

	if err := handler.Handle(conn); err != nil {
		res = &common.StatusResp{
			Tag: cmd.Tag,
			Type: common.NO,
			Info: err.Error(),
		}
	} else {
		res = &common.StatusResp{
			Tag: cmd.Tag,
			Type: common.OK,
			Info: cmd.Name + " completed",
		}
	}

	return
}

// Create a new IMAP server from an existing listener.
func NewServer(l net.Listener, bkd backend.Backend) *Server {
	s := &Server{
		listener: l,
		Backend: bkd,
	}

	s.auths = map[string]sasl.Server{
		"PLAIN": sasl.NewPlainServer(bkd),
	}

	s.commands = map[string]HandlerFactory{
		common.Noop: func () Handler { return &Noop{} },
		common.Capability: func () Handler { return &Capability{} },
		common.Logout: func () Handler { return &Logout{} },

		common.Login: func () Handler { return &Login{} },
		common.Authenticate: func () Handler { return &Authenticate{Mechanisms: s.auths} },

		common.Select: func () Handler { return &Select{} },
		common.Examine: func () Handler {
			hdlr := &Select{}
			hdlr.ReadOnly = true
			return hdlr
		},
		common.List: func () Handler { return &List{} },
		common.Status: func () Handler { return &Status{} },
	}

	go s.listen()
	return s
}

func Listen(addr string, bkd backend.Backend) (s *Server, err error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}

	s = NewServer(l, bkd)
	return
}
