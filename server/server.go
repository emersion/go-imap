package server

import (
	"errors"
	"io"
	"log"
	"net"

	imap "github.com/emersion/imap/common"
)

type Handler interface {
	imap.Parser

	Handle(conn *Conn) error
}

type HandlerFactory func () Handler

type Server struct {
	listener net.Listener
	commands map[string]HandlerFactory
}

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
		fields, err := conn.ReadLine()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Println("Error reading command:", err)
			continue
		}

		var res imap.WriterTo

		cmd := &imap.Command{}
		if err := cmd.Parse(fields); err != nil {
			res = &imap.StatusResp{
				Tag: "*",
				Type: imap.BAD,
				Info: err.Error(),
			}
		} else {
			res, err = s.handleCommand(cmd, conn)
			if err != nil {
				res = &imap.StatusResp{
					Tag: cmd.Tag,
					Type: imap.BAD,
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

func (s *Server) handleCommand(cmd *imap.Command, conn *Conn) (res imap.WriterTo, err error) {
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
		res = &imap.StatusResp{
			Tag: cmd.Tag,
			Type: imap.NO,
			Info: err.Error(),
		}
	} else {
		res = &imap.StatusResp{
			Tag: cmd.Tag,
			Type: imap.OK,
			Info: cmd.Name + " completed",
		}
	}

	return
}

func NewServer(l net.Listener) *Server {
	s := &Server{
		listener: l,
		commands: map[string]HandlerFactory{
			imap.Noop: func () Handler { return &Noop{} },
			imap.Capability: func () Handler { return &Capability{} },
		},
	}

	go s.listen()
	return s
}

func Listen(addr string) (s *Server, err error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}

	s = NewServer(l)
	return
}
