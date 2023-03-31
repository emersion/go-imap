// Package imapserver implements an IMAP server.
package imapserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

var errClosed = errors.New("imapserver: server closed")

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

	listenerWaitGroup sync.WaitGroup

	mutex     sync.Mutex
	listeners map[net.Listener]struct{}
	conns     map[*Conn]struct{}
	closed    bool
}

// New creates a new server.
func New(options *Options) *Server {
	return &Server{
		options:   *options,
		listeners: make(map[net.Listener]struct{}),
		conns:     make(map[*Conn]struct{}),
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
	s.mutex.Lock()
	ok := !s.closed
	if ok {
		s.listeners[ln] = struct{}{}
	}
	s.mutex.Unlock()
	if !ok {
		return errClosed
	}

	defer func() {
		s.mutex.Lock()
		delete(s.listeners, ln)
		s.mutex.Unlock()
	}()

	s.listenerWaitGroup.Add(1)
	defer s.listenerWaitGroup.Done()

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

// Close immediately closes all active listeners and connections.
//
// Close returns any error returned from closing the server's underlying
// listeners.
//
// Once Close has been called on a server, it may not be reused; future calls
// to methods such as Serve will return an error.
func (s *Server) Close() error {
	var err error

	s.mutex.Lock()
	ok := !s.closed
	if ok {
		s.closed = true
		for l := range s.listeners {
			if closeErr := l.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
	}
	s.mutex.Unlock()
	if !ok {
		return errClosed
	}

	s.listenerWaitGroup.Wait()

	s.mutex.Lock()
	for c := range s.conns {
		c.conn.Close()
	}
	s.mutex.Unlock()

	return err
}
