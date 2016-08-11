// An IMAP server.
package server

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/responses"
	"github.com/emersion/go-sasl"
)

// A command handler.
type Handler interface {
	imap.Parser

	// Handle this command for a given connection.
	//
	// By default, after this function has returned a status response is sent. To
	// prevent this behavior handlers can use ErrStatusResp or ErrNoStatusResp.
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
type SaslServerFactory func(conn Conn) sasl.Server

// An IMAP extension.
type Extension interface {
	// Get capabilities provided by this extension for a given connection state.
	Capabilities(state imap.ConnState) []string
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

type errStatusResp struct {
	resp *imap.StatusResp
}

func (err *errStatusResp) Error() string {
	return ""
}

// ErrStatusResp can be returned by a Handler to replace the default status
// response. The response tag must be empty.
//
// To disable the default status response, use ErrNoStatusResp instead.
func ErrStatusResp(res *imap.StatusResp) error {
	return &errStatusResp{res}
}

// ErrNoStatusResp can be returned by a Handler to prevent the default status
// response from being sent.
func ErrNoStatusResp() error {
	return &errStatusResp{nil}
}

// An IMAP server.
type Server struct {
	listener net.Listener
	conns []Conn

	commands map[string]HandlerFactory
	auths map[string]SaslServerFactory
	extensions []Extension

	// TCP address to listen on.
	Addr string
	// This server's TLS configuration.
	TLSConfig *tls.Config
	// This server's backend.
	Backend backend.Backend
	// Backend updates that will be sent to connected clients.
	Updates *backend.Updates
	// Allow authentication over unencrypted connections.
	AllowInsecureAuth bool
	// Print all network activity to STDOUT.
	Debug bool
}

// Create a new IMAP server from an existing listener.
func New(bkd backend.Backend) *Server {
	s := &Server{
		Backend: bkd,
	}

	s.auths = map[string]SaslServerFactory{
		"PLAIN": func(conn Conn) sasl.Server {
			return sasl.NewPlainServer(func(identity, username, password string) error {
				if identity != "" && identity != username {
					return errors.New("Identities not supported")
				}

				user, err := bkd.Login(username, password)
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
		imap.Noop: func() Handler { return &Noop{} },
		imap.Capability: func() Handler { return &Capability{} },
		imap.Logout: func() Handler { return &Logout{} },

		imap.StartTLS: func() Handler { return &StartTLS{} },
		imap.Login: func() Handler { return &Login{} },
		imap.Authenticate: func() Handler { return &Authenticate{} },

		imap.Select: func() Handler { return &Select{} },
		imap.Examine: func() Handler {
			hdlr := &Select{}
			hdlr.ReadOnly = true
			return hdlr
		},
		imap.Create: func() Handler { return &Create{} },
		imap.Delete: func() Handler { return &Delete{} },
		imap.Rename: func() Handler { return &Rename{} },
		imap.Subscribe: func() Handler { return &Subscribe{} },
		imap.Unsubscribe: func() Handler { return &Unsubscribe{} },
		imap.List: func() Handler { return &List{} },
		imap.Lsub: func() Handler {
			hdlr := &List{}
			hdlr.Subscribed = true
			return hdlr
		},
		imap.Status: func() Handler { return &Status{} },
		imap.Append: func() Handler { return &Append{} },

		imap.Check: func() Handler { return &Check{} },
		imap.Close: func() Handler { return &Close{} },
		imap.Expunge: func() Handler { return &Expunge{} },
		imap.Search: func() Handler { return &Search{} },
		imap.Fetch: func() Handler { return &Fetch{} },
		imap.Store: func() Handler { return &Store{} },
		imap.Copy: func() Handler { return &Copy{} },
		imap.Uid: func() Handler { return &Uid{} },
	}

	return s
}

// Serve accepts incoming connections on the Listener l.
func (s *Server) Serve(l net.Listener) error {
	s.listener = l
	defer s.Close()

	go s.listenUpdates()

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}

		var conn Conn = newConn(s, c)
		if s.Debug {
			conn.conn().SetDebug(true)
		}

		for _, ext := range s.extensions {
			if ext, ok := ext.(ConnExtension); ok {
				conn = ext.NewConn(conn)
			}
		}

		go s.handleConn(conn)
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

func (s *Server) handleConn(conn Conn) error {
	s.conns = append(s.conns, conn)
	defer (func() {
		conn.Close()

		for i, c := range s.conns {
			if c == conn {
				s.conns = append(s.conns[:i], s.conns[i+1:]...)
				break
			}
		}
	})()

	// Send greeting
	if err := conn.conn().greet(); err != nil {
		return err
	}

	for {
		if conn.Context().State == imap.LogoutState {
			return nil
		}

		var res *imap.StatusResp
		var up Upgrader

		conn.conn().Wait()
		fields, err := conn.conn().ReadLine()
		if err == io.EOF || conn.Context().State == imap.LogoutState {
			return nil
		}
		if err != nil {
			if imap.IsParseError(err) {
				res = &imap.StatusResp{
					Type: imap.StatusBad,
					Info: err.Error(),
				}
			} else {
				log.Println("Error reading command:", err)
				return err
			}
		} else {
			cmd := &imap.Command{}
			if err := cmd.Parse(fields); err != nil {
				res = &imap.StatusResp{
					Tag: cmd.Tag,
					Type: imap.StatusBad,
					Info: err.Error(),
				}
			} else {
				var err error
				res, up, err = s.handleCommand(cmd, conn)
				if err != nil {
					res = &imap.StatusResp{
						Tag: cmd.Tag,
						Type: imap.StatusBad,
						Info: err.Error(),
					}
				}
			}
		}

		if res != nil {
			if err := conn.WriteResp(res); err != nil {
				log.Println("Error writing response:", err)
				continue
			}

			if up != nil && res.Type == imap.StatusOk {
				if err := up.Upgrade(conn); err != nil {
					log.Println("Error upgrading connection:", err)
					return err
				}
			}
		}
	}
}

// Get a command handler factory for the provided command name.
func (s *Server) Command(name string) HandlerFactory {
	// Extensions can override builtin commands
	for _, ext := range s.extensions {
		if h := ext.Command(name); h != nil {
			return h
		}
	}

	return s.commands[name]
}

func (s *Server) commandHandler(cmd *imap.Command) (hdlr Handler, err error) {
	newHandler := s.Command(cmd.Name)
	if newHandler == nil {
		err = errors.New("Unknown command")
		return
	}

	hdlr = newHandler()
	err = hdlr.Parse(cmd.Arguments)
	return
}

func (s *Server) handleCommand(cmd *imap.Command, conn Conn) (res *imap.StatusResp, up Upgrader, err error) {
	hdlr, err := s.commandHandler(cmd)
	if err != nil {
		return
	}

	hdlrErr := hdlr.Handle(conn)
	if statusErr, ok := hdlrErr.(*errStatusResp); ok {
		res = statusErr.resp
	} else if hdlrErr != nil {
		res = &imap.StatusResp{
			Type: imap.StatusNo,
			Info: hdlrErr.Error(),
		}
	} else {
		res = &imap.StatusResp{
			Type: imap.StatusOk,
		}
	}

	if res != nil {
		res.Tag = cmd.Tag

		if res.Type == imap.StatusOk && res.Info == "" {
			res.Info = cmd.Name + " completed"
		}
	}

	up, _ = hdlr.(Upgrader)
	return
}

func (s *Server) listenUpdates() (err error) {
	updater, ok := s.Backend.(backend.Updater)
	if !ok {
		return
	}
	s.Updates = updater.Updates()

	var update *backend.Update
	var res imap.WriterTo
	for {
		// TODO: do not generate response if nobody will receive it

		select {
		case status := <-s.Updates.Statuses:
			update = &status.Update
			res = status.StatusResp
		case mailbox := <-s.Updates.Mailboxes:
			update = &mailbox.Update
			res = &responses.Select{Mailbox: mailbox.MailboxStatus}
		case message := <-s.Updates.Messages:
			update = &message.Update

			ch := make(chan *imap.Message, 1)
			ch <- message.Message
			close(ch)

			res = &responses.Fetch{Messages: ch}
		case expunge := <-s.Updates.Expunges:
			update = &expunge.Update

			ch := make(chan uint32, 1)
			ch <- expunge.SeqNum
			close(ch)

			res = &responses.Expunge{SeqNums: ch}
		}

		// Format response
		b := &bytes.Buffer{}
		w := imap.NewWriter(b)
		if err := res.WriteTo(w); err != nil {
			log.Println("WARN: cannot format unlateral update:", err)
		}

		for _, conn := range s.conns {
			ctx := conn.Context()

			if update.Username != "" && (ctx.User == nil || ctx.User.Username() != update.Username) {
				continue
			}
			if update.Mailbox != "" && (ctx.Mailbox == nil || ctx.Mailbox.Name() != update.Mailbox) {
				continue
			}
			if conn.conn().silent {
				// If silent is set, do not send message updates
				if _, ok := res.(*responses.Fetch); ok {
					continue
				}
			}

			conn.conn().locker.Lock()
			if _, err := conn.conn().Writer.Write(b.Bytes()); err != nil {
				log.Println("WARN: error sending unilateral update:", err)
			}
			conn.conn().Flush()
			conn.conn().locker.Unlock()
		}

		if update.Done != nil {
			close(update.Done)
		}
	}
}

// Get this server's capabilities for the provided connection state.
func (s *Server) Capabilities(currentState imap.ConnState) (caps []string) {
	caps = []string{"IMAP4rev1"}

	for _, ext := range s.extensions {
		caps = append(caps, ext.Capabilities(currentState)...)
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

// Enable an IMAP extension on this server.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (s *Server) Enable(extension Extension) {
	s.extensions = append(s.extensions, extension)
}

// Enable an authentication mechanism on this server.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (s *Server) EnableAuth(name string, f SaslServerFactory) {
	s.auths[name] = f
}
