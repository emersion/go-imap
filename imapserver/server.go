// Package imapserver implements an IMAP server.
package imapserver

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

type Logger interface {
	Printf(format string, args ...interface{})
}

type Server struct {
	NewSession func() (Session, error)
	Logger     Logger
}

func (s *Server) logger() Logger {
	if s.Logger == nil {
		return log.Default()
	}
	return s.Logger
}

func (s *Server) Serve(ln net.Listener) error {
	var delay time.Duration
	for {
		conn, err := ln.Accept()
		if ne, ok := err.(net.Error); ok && ne.Temporary() {
			if delay == 0 {
				delay = 5 * time.Millisecond
			} else {
				delay *= 2
			}
			if max := 1 * time.Second; delay > max {
				delay = max
			}
			s.logger().Printf("accept error (retrying in %v): %v", delay, err)
			time.Sleep(delay)
			continue
		} else if errors.Is(err, net.ErrClosed) {
			return nil
		} else if err != nil {
			return fmt.Errorf("accept error: %w", err)
		}

		delay = 0
		go newConn(conn, s).serve()
	}
}
