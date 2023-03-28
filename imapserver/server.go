// Package imapserver implements an IMAP server.
package imapserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

// Logger is a facility to log error messages.
type Logger interface {
	Printf(format string, args ...interface{})
}

// Options contains server options.
//
// The only required field is NewSession.
type Options struct {
	// NewSession is called when a client connects.
	NewSession func(*Conn) (Session, error)
	// Logger is a logger to print error messages. If nil, log.Default is used.
	Logger Logger
	// TLSConfig is a TLS configuration for STARTTLS. If nil, STARTTLS is
	// disabled.
	TLSConfig *tls.Config
	// InsecureAuth allows clients to authenticate without TLS. In this mode,
	// the server is susceptible to man-in-the-middle attacks.
	InsecureAuth bool
	// Raw ingress and egress data will be written to this writer, if any.
	// Note, this may include sensitive information such as credentials used
	// during authentication.
	DebugWriter io.Writer
}

func (options *Options) wrapReadWriter(rw io.ReadWriter) io.ReadWriter {
	if options.DebugWriter == nil {
		return rw
	}
	return struct {
		io.Reader
		io.Writer
	}{
		Reader: io.TeeReader(rw, options.DebugWriter),
		Writer: io.MultiWriter(rw, options.DebugWriter),
	}
}

// Server is an IMAP server.
type Server struct {
	options Options
}

// New creates a new server.
func New(options *Options) *Server {
	return &Server{
		options: *options,
	}
}

func (s *Server) logger() Logger {
	if s.options.Logger == nil {
		return log.Default()
	}
	return s.options.Logger
}

// Serve accepts incoming connections on the listener ln.
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
