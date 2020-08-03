// Package server provides an IMAP server.
package server

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-sasl"
)

// The minimum autologout duration defined in RFC 3501 section 5.4.
const MinAutoLogout = 30 * time.Minute

// A command handler.
type Handler interface {
	imap.Parser

	// Handle this command for a given connection.
	//
	// By default, after this function has returned a status response is sent. To
	// prevent this behavior handlers can use imap.ErrStatusResp.
	Handle(conn Conn) error
}

// A connection upgrader. If a Handler is also an Upgrader, the connection will
// be upgraded after the Handler succeeds.
//
// This should only be used by libraries implementing an IMAP extension (e.g.
// COMPRESS).
type Upgrader interface {
	// Upgrade the connection. This method should call conn.Upgrade().
	Upgrade(conn Conn) error
}

// A function that creates handlers.
type HandlerFactory func() Handler

// A function that creates SASL servers.
type SASLServerFactory func(conn Conn) sasl.Server

// An IMAP extension.
type Extension interface {
	// Get capabilities provided by this extension for a given connection.
	Capabilities(c Conn) []string
	// Get the command handler factory for the provided command name.
	Command(name string) HandlerFactory
}

// An extension that provides additional features to each connection.
type ConnExtension interface {
	Extension

	// This function will be called when a client connects to the server. It can
	// be used to add new features to the default Conn interface by implementing
	// new methods.
	NewConn(c Conn) Conn
}

// ErrStatusResp can be returned by a Handler to replace the default status
// response. The response tag must be empty.
//
// Deprecated: Use imap.ErrStatusResp{res} instead.
//
// To disable the default status response, use imap.ErrStatusResp{nil} instead.
func ErrStatusResp(res *imap.StatusResp) error {
	return &imap.ErrStatusResp{res}
}

// ErrNoStatusResp can be returned by a Handler to prevent the default status
// response from being sent.
//
// Deprecated: Use imap.ErrStatusResp{nil} instead
func ErrNoStatusResp() error {
	return &imap.ErrStatusResp{nil}
}

// An IMAP server.
type Server struct {
	locker    sync.Mutex
	listeners map[net.Listener]struct{}
	conns     map[Conn]struct{}

	commands    map[string]HandlerFactory
	auths       map[string]SASLServerFactory
	extensions  []Extension
	backendExts map[backend.Extension]struct{}

	// TCP address to listen on.
	Addr string
	// This server's TLS configuration.
	TLSConfig *tls.Config
	// This server's backend.
	Backend backend.Backend
	// Automatically logout clients after a duration. To do not logout users
	// automatically, set this to zero. The duration MUST be at least
	// MinAutoLogout (as stated in RFC 3501 section 5.4).
	AutoLogout time.Duration
	// Allow authentication over unencrypted connections.
	AllowInsecureAuth bool
	// An io.Writer to which all network activity will be mirrored.
	Debug io.Writer
	// ErrorLog specifies an optional logger for errors accepting
	// connections and unexpected behavior from handlers.
	// If nil, logging goes to os.Stderr via the log package's
	// standard logger.
	ErrorLog imap.Logger
	// The maximum literal size, in bytes. Literals exceeding this size will be
	// rejected. A value of zero disables the limit (this is the default).
	MaxLiteralSize uint32
}

// Create a new IMAP server from an existing listener.
func New(bkd backend.Backend) *Server {
	s := &Server{
		listeners:   make(map[net.Listener]struct{}),
		conns:       make(map[Conn]struct{}),
		backendExts: map[backend.Extension]struct{}{},
		Backend:     bkd,
		ErrorLog:    log.New(os.Stderr, "imap/server: ", log.LstdFlags),
	}

	for _, ext := range bkd.SupportedExtensions() {
		s.backendExts[ext] = struct{}{}
	}

	s.auths = map[string]SASLServerFactory{
		sasl.Plain: func(conn Conn) sasl.Server {
			return sasl.NewPlainServer(func(identity, username, password string) error {
				if identity != "" && identity != username {
					return errors.New("Identities not supported")
				}

				user, err := bkd.Login(conn.Info(), username, password)
				if err != nil {
					return err
				}

				ctx := conn.Context()
				ctx.State = imap.AuthenticatedState
				ctx.User = user
				return nil
			})
		},
	}

	s.commands = map[string]HandlerFactory{
		"NOOP":       func() Handler { return &Noop{} },
		"CAPABILITY": func() Handler { return &Capability{} },
		"LOGOUT":     func() Handler { return &Logout{} },

		"STARTTLS":     func() Handler { return &StartTLS{} },
		"LOGIN":        func() Handler { return &Login{} },
		"AUTHENTICATE": func() Handler { return &Authenticate{} },

		"SELECT": func() Handler { return &Select{} },
		"EXAMINE": func() Handler {
			hdlr := &Select{}
			hdlr.ReadOnly = true
			return hdlr
		},
		"CREATE":      func() Handler { return &Create{} },
		"DELETE":      func() Handler { return &Delete{} },
		"RENAME":      func() Handler { return &Rename{} },
		"SUBSCRIBE":   func() Handler { return &Subscribe{} },
		"UNSUBSCRIBE": func() Handler { return &Unsubscribe{} },
		"LIST":        func() Handler { return &List{} },
		"LSUB": func() Handler {
			hdlr := &List{}
			hdlr.Subscribed = true
			return hdlr
		},
		"STATUS": func() Handler { return &Status{} },
		"APPEND": func() Handler { return &Append{} },

		"CHECK":   func() Handler { return &Check{} },
		"CLOSE":   func() Handler { return &Close{} },
		"EXPUNGE": func() Handler { return &Expunge{} },
		"SEARCH":  func() Handler { return &Search{} },
		"FETCH":   func() Handler { return &Fetch{} },
		"STORE":   func() Handler { return &Store{} },
		"COPY":    func() Handler { return &Copy{} },
		"UID":     func() Handler { return &Uid{} },
	}

	return s
}

// Serve accepts incoming connections on the Listener l.
func (s *Server) Serve(l net.Listener) error {
	s.locker.Lock()
	s.listeners[l] = struct{}{}
	s.locker.Unlock()

	defer func() {
		s.locker.Lock()
		defer s.locker.Unlock()
		l.Close()
		delete(s.listeners, l)
	}()

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}

		var conn Conn = newConn(s, c)
		for _, ext := range s.extensions {
			if ext, ok := ext.(ConnExtension); ok {
				conn = ext.NewConn(conn)
			}
		}

		go s.serveConn(conn)
	}
}

// ListenAndServe listens on the TCP network address s.Addr and then calls Serve
// to handle requests on incoming connections.
//
// If s.Addr is blank, ":imap" is used.
func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":imap"
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.Serve(l)
}

// ListenAndServeTLS listens on the TCP network address s.Addr and then calls
// Serve to handle requests on incoming TLS connections.
//
// If s.Addr is blank, ":imaps" is used.
func (s *Server) ListenAndServeTLS() error {
	addr := s.Addr
	if addr == "" {
		addr = ":imaps"
	}

	l, err := tls.Listen("tcp", addr, s.TLSConfig)
	if err != nil {
		return err
	}

	return s.Serve(l)
}

func (s *Server) serveConn(conn Conn) error {
	s.locker.Lock()
	s.conns[conn] = struct{}{}
	s.locker.Unlock()

	defer func() {
		s.locker.Lock()
		defer s.locker.Unlock()
		conn.Close()
		delete(s.conns, conn)
	}()

	return conn.serve(conn)
}

// Command gets a command handler factory for the provided command name.
func (s *Server) Command(name string) HandlerFactory {
	// Extensions can override builtin commands
	for _, ext := range s.extensions {
		if h := ext.Command(name); h != nil {
			return h
		}
	}

	return s.commands[name]
}

// ForEachConn iterates through all opened connections.
func (s *Server) ForEachConn(f func(Conn)) {
	s.locker.Lock()
	defer s.locker.Unlock()
	for conn := range s.conns {
		f(conn)
	}
}

// Stops listening and closes all current connections.
func (s *Server) Close() error {
	s.locker.Lock()
	defer s.locker.Unlock()

	for l := range s.listeners {
		l.Close()
	}

	for conn := range s.conns {
		conn.Close()
	}

	return nil
}

// Enable some IMAP extensions on this server.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (s *Server) Enable(extensions ...Extension) {
	s.extensions = append(s.extensions, extensions...)
}

// Enable an authentication mechanism on this server.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (s *Server) EnableAuth(name string, f SASLServerFactory) {
	s.auths[name] = f
}
