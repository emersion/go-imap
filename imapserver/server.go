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

	"github.com/emersion/go-imap/v2"
)

var errClosed = errors.New("imapserver: server closed")

// Logger is a facility to log error messages.
type Logger interface {
	Printf(format string, args ...interface{})
}

// SupportedCaps describes capabilities supported by the server.
type SupportedCaps struct {
	// IMAP protocol version
	IMAP4rev1 bool // RFC 3501
	IMAP4rev2 bool // RFC 9051

	// Capabilities which are part of IMAP4rev2 and need to be explicitly
	// enabled by IMAP4rev1-only servers
	Namespace    bool // RFC 2342
	UIDPlus      bool // RFC 4315
	ESearch      bool // RFC 4731
	SearchRes    bool // RFC 5182
	ListExtended bool // RFC 5258
	ListStatus   bool // RFC 5819
	Move         bool // RFC 6851
	StatusSize   bool // RFC 8438
	Binary       bool // RFC 3516
	Children     bool // RFC 3348

	// Capabilities which need to be explicitly enabled on both IMAP4rev1 and
	// IMAP4rev2 servers
	SpecialUse       bool // RFC 6154
	CreateSpecialUse bool // RFC 6154
	LiteralPlus      bool // RFC 7888
	Unauthenticate   bool // RFC 8437
	AppendLimit      bool // RFC 7889
}

func (caps *SupportedCaps) set() imap.CapSet {
	m := map[imap.Cap]bool{
		imap.CapIMAP4rev1:        caps.IMAP4rev1,
		imap.CapIMAP4rev2:        caps.IMAP4rev2,
		imap.CapNamespace:        caps.Namespace,
		imap.CapUIDPlus:          caps.UIDPlus,
		imap.CapESearch:          caps.ESearch,
		imap.CapSearchRes:        caps.SearchRes,
		imap.CapListExtended:     caps.ListExtended,
		imap.CapListStatus:       caps.ListStatus,
		imap.CapMove:             caps.Move,
		imap.CapStatusSize:       caps.StatusSize,
		imap.CapBinary:           caps.Binary,
		imap.CapChildren:         caps.Children,
		imap.CapSpecialUse:       caps.SpecialUse,
		imap.CapCreateSpecialUse: caps.CreateSpecialUse,
		imap.CapLiteralPlus:      caps.LiteralPlus,
		imap.CapUnauthenticate:   caps.Unauthenticate,
		imap.CapAppendLimit:      caps.AppendLimit,
	}

	set := make(imap.CapSet, len(m))
	for name, ok := range m {
		if ok {
			set[name] = struct{}{}
		}
	}
	return set
}

// Options contains server options.
//
// The only required field is NewSession.
type Options struct {
	// NewSession is called when a client connects.
	NewSession func(*Conn) (Session, *GreetingData, error)
	// Supported capabilities. If nil, only IMAP4rev1 is advertised. This set
	// must contain at least IMAP4rev1 or IMAP4rev2.
	Caps *SupportedCaps
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

func (options *Options) caps() *SupportedCaps {
	if options.Caps != nil {
		return options.Caps
	}
	return &SupportedCaps{IMAP4rev1: true}
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
	if caps := options.caps().set(); !caps.Has(imap.CapIMAP4rev2) && !caps.Has(imap.CapIMAP4rev1) {
		panic("imapserver: at least IMAP4rev1 must be supported")
	}
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

// ListenAndServe listens on the TCP network address addr and then calls Serve.
//
// If addr is empty, ":143" is used.
func (s *Server) ListenAndServe(addr string) error {
	if addr == "" {
		addr = ":143"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// ListenAndServeTLS listens on the TCP network address addr and then calls
// Serve to handle incoming TLS connections.
//
// The TLS configuration set in Options.TLSConfig is used. If addr is empty,
// ":993" is used.
func (s *Server) ListenAndServeTLS(addr string) error {
	if addr == "" {
		addr = ":993"
	}
	ln, err := tls.Listen("tcp", addr, s.options.TLSConfig)
	if err != nil {
		return err
	}
	return s.Serve(ln)
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
		c.mutex.Lock()
		c.conn.Close()
		c.mutex.Unlock()
	}
	s.mutex.Unlock()

	return err
}
