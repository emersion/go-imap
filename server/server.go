// An IMAP server.
package server

import (
	"crypto/tls"
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
	conns []*Conn

	caps map[string]common.ConnState
	commands map[string]HandlerFactory
	auths map[string]sasl.Server

	// This server's backend.
	Backend backend.Backend
	// This server's TLS configuration.
	TLSConfig *tls.Config
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

		conn := newConn(s, c)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn *Conn) error {
	s.conns = append(s.conns, conn)
	defer conn.Close()

	// Send greeting
	if err := conn.greet(); err != nil {
		return err
	}

	for {
		if conn.State == common.LogoutState {
			return nil
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

func (s *Server) getCommandHandler(cmd *common.Command) (hdlr Handler, err error) {
	newHandler, ok := s.commands[cmd.Name]
	if !ok {
		err = errors.New("Unknown command")
		return
	}

	hdlr = newHandler()
	err = hdlr.Parse(cmd.Arguments)
	return
}

func (s *Server) handleCommand(cmd *common.Command, conn *Conn) (res common.WriterTo, err error) {
	hdlr, err := s.getCommandHandler(cmd)
	if err != nil {
		return
	}

	if err := hdlr.Handle(conn); err != nil {
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

func (s *Server) getCaps(currentState common.ConnState) (caps []string) {
	for name, state := range s.caps {
		if currentState & state != 0 {
			caps = append(caps, name)
		}
	}
	return
}

// Stops listening and closes all current connections.
func (s *Server) Close() error {
	if err := s.listener.Close(); err != nil {
		return err
	}

	for _, conn := range s.conns {
		conn.Close()
	}

	return nil
}

// Create a new IMAP server from an existing listener.
func NewServer(l net.Listener, bkd backend.Backend) *Server {
	s := &Server{
		listener: l,
		caps: map[string]common.ConnState{},
		Backend: bkd,
	}

	s.auths = map[string]sasl.Server{
		"PLAIN": sasl.NewPlainServer(bkd),
	}

	s.commands = map[string]HandlerFactory{
		common.Noop: func () Handler { return &Noop{} },
		common.Capability: func () Handler { return &Capability{} },
		common.Logout: func () Handler { return &Logout{} },

		common.StartTLS: func () Handler { return &StartTLS{} },
		common.Login: func () Handler { return &Login{} },
		common.Authenticate: func () Handler { return &Authenticate{Mechanisms: s.auths} },

		common.Select: func () Handler { return &Select{} },
		common.Examine: func () Handler {
			hdlr := &Select{}
			hdlr.ReadOnly = true
			return hdlr
		},
		common.Create: func () Handler { return &Create{} },
		common.Delete: func () Handler { return &Delete{} },
		common.Rename: func () Handler { return &Rename{} },
		common.List: func () Handler { return &List{} },
		common.Status: func () Handler { return &Status{} },
		common.Append: func () Handler { return &Append{} },

		common.Check: func () Handler { return &Check{} },
		common.Close: func () Handler { return &Close{} },
		common.Expunge: func () Handler { return &Expunge{} },
		common.Search: func () Handler { return &Search{} },
		common.Fetch: func () Handler { return &Fetch{} },
		common.Store: func () Handler { return &Store{} },
		common.Copy: func () Handler { return &Copy{} },
		common.Uid: func () Handler { return &Uid{} },
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

func ListenTLS(addr string, bkd backend.Backend, tlsConfig *tls.Config) (s *Server, err error) {
	l, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return
	}

	s = NewServer(l, bkd)
	return
}
