// An IMAP server.
package server

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"

	"github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/responses"
	"github.com/emersion/go-sasl"
)

// A command handler.
type Handler interface {
	common.Parser

	// Handle this command for a given connection.
	//
	// By default, after this function has returned a status response is sent. To
	// prevent this behavior handlers can use ErrStatusResp or ErrNoStatusResp.
	Handle(conn *Conn) error
}

// A connection upgrader. If a Handler is also an Upgrader, the connection will
// be upgraded after the Handler succeeds.
//
// This should only be used by libraries implementing an IMAP extension (e.g.
// COMPRESS).
type Upgrader interface {
	// Upgrade the connection. This method should call conn.Upgrade().
	Upgrade(conn *Conn) error
}

// A function that creates handlers.
type HandlerFactory func() Handler

// A function that creates SASL servers.
type SaslServerFactory func(conn *Conn) sasl.Server

type errStatusResp struct {
	resp *common.StatusResp
}

func (err *errStatusResp) Error() string {
	return ""
}

// ErrStatusResp can be returned by a Handler to replace the default status
// response. The response tag must be empty.
//
// To disable the default status response, use ErrNoStatusResp instead.
func ErrStatusResp(res *common.StatusResp) error {
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
	conns []*Conn

	caps map[string]common.ConnState
	commands map[string]HandlerFactory
	auths map[string]SaslServerFactory

	// This server's backend.
	Backend backend.Backend
	// Backend updates that will be sent to connected clients.
	Updates *backend.Updates
	// This server's TLS configuration.
	TLSConfig *tls.Config
	// Allow authentication over unencrypted connections.
	AllowInsecureAuth bool
	// Print all network activity to STDOUT.
	Debug bool
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
		if s.Debug {
			conn.SetDebug(true)
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn *Conn) error {
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
	if err := conn.greet(); err != nil {
		return err
	}

	for {
		if conn.State == common.LogoutState {
			return nil
		}

		conn.Wait()

		fields, err := conn.ReadLine()
		if err == io.EOF || conn.State == common.LogoutState {
			return nil
		}
		if err != nil {
			log.Println("Error reading command:", err)
			return err
		}

		var res *common.StatusResp
		var up Upgrader

		cmd := &common.Command{}
		if err := cmd.Parse(fields); err != nil {
			res = &common.StatusResp{
				Tag: cmd.Tag,
				Type: common.StatusBad,
				Info: err.Error(),
			}
		} else {
			var err error
			res, up, err = s.handleCommand(cmd, conn)
			if err != nil {
				res = &common.StatusResp{
					Tag: cmd.Tag,
					Type: common.StatusBad,
					Info: err.Error(),
				}
			}
		}

		if res != nil {
			if err := conn.WriteResp(res); err != nil {
				log.Println("Error writing response:", err)
				continue
			}
		}

		if up != nil && res.Type == common.StatusOk {
			if err := up.Upgrade(conn); err != nil {
				log.Println("Error upgrading connection:", err)
				return err
			}
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

func (s *Server) handleCommand(cmd *common.Command, conn *Conn) (res *common.StatusResp, up Upgrader, err error) {
	hdlr, err := s.getCommandHandler(cmd)
	if err != nil {
		return
	}

	hdlrErr := hdlr.Handle(conn)
	if statusErr, ok := hdlrErr.(*errStatusResp); ok {
		res = statusErr.resp
	} else if hdlrErr != nil {
		res = &common.StatusResp{
			Type: common.StatusNo,
			Info: hdlrErr.Error(),
		}
	} else {
		res = &common.StatusResp{
			Type: common.StatusOk,
		}
	}

	if res != nil {
		res.Tag = cmd.Tag

		if res.Type == common.StatusOk && res.Info == "" {
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
	var res common.WriterTo
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

			ch := make(chan *common.Message, 1)
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
		w := common.NewWriter(b)
		if err := res.WriteTo(w); err != nil {
			log.Println("WARN: cannot format unlateral update:", err)
		}

		for _, conn := range s.conns {
			if update.Username != "" && (conn.User == nil || conn.User.Username() != update.Username) {
				continue
			}
			if update.Mailbox != "" && (conn.Mailbox == nil || conn.Mailbox.Name() != update.Mailbox) {
				continue
			}
			if conn.silent {
				// If silent is set, do not send message updates
				if _, ok := res.(*responses.Fetch); ok {
					continue
				}
			}

			conn.locker.Lock()
			if _, err := conn.Writer.Write(b.Bytes()); err != nil {
				log.Println("WARN: error sending unilateral update:", err)
			}
			conn.Flush()
			conn.locker.Unlock()
		}

		if update.Done != nil {
			close(update.Done)
		}
	}
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

// Register a new capability that will be advertised by this server.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (s *Server) RegisterCapability(name string, state common.ConnState) {
	s.caps[name] = state
}

// Register a new authentication mechanism for this server.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (s *Server) RegisterAuth(name string, f SaslServerFactory) {
	s.auths[name] = f
}

// Register a new command for this server.
//
// This function should not be called directly, it must only be used by
// libraries implementing extensions of the IMAP protocol.
func (s *Server) RegisterCommand(name string, f HandlerFactory) {
	s.commands[name] = f
}

// Create a new IMAP server from an existing listener.
func NewServer(l net.Listener, bkd backend.Backend) *Server {
	s := &Server{
		listener: l,
		caps: map[string]common.ConnState{},
		Backend: bkd,
	}

	s.auths = map[string]SaslServerFactory{
		"PLAIN": func(conn *Conn) sasl.Server {
			return sasl.NewPlainServer(func(identity, username, password string) error {
				if identity != "" && identity != username {
					return errors.New("Identities not supported")
				}

				user, err := bkd.Login(username, password)
				if err != nil {
					return err
				}

				conn.State = common.AuthenticatedState
				conn.User = user
				return nil
			})
		},
	}

	s.commands = map[string]HandlerFactory{
		common.Noop: func() Handler { return &Noop{} },
		common.Capability: func() Handler { return &Capability{} },
		common.Logout: func() Handler { return &Logout{} },

		common.StartTLS: func() Handler { return &StartTLS{} },
		common.Login: func() Handler { return &Login{} },
		common.Authenticate: func() Handler { return &Authenticate{} },

		common.Select: func() Handler { return &Select{} },
		common.Examine: func() Handler {
			hdlr := &Select{}
			hdlr.ReadOnly = true
			return hdlr
		},
		common.Create: func() Handler { return &Create{} },
		common.Delete: func() Handler { return &Delete{} },
		common.Rename: func() Handler { return &Rename{} },
		common.Subscribe: func() Handler { return &Subscribe{} },
		common.Unsubscribe: func() Handler { return &Unsubscribe{} },
		common.List: func() Handler { return &List{} },
		common.Lsub: func() Handler {
			hdlr := &List{}
			hdlr.Subscribed = true
			return hdlr
		},
		common.Status: func() Handler { return &Status{} },
		common.Append: func() Handler { return &Append{} },

		common.Check: func() Handler { return &Check{} },
		common.Close: func() Handler { return &Close{} },
		common.Expunge: func() Handler { return &Expunge{} },
		common.Search: func() Handler { return &Search{} },
		common.Fetch: func() Handler { return &Fetch{} },
		common.Store: func() Handler { return &Store{} },
		common.Copy: func() Handler { return &Copy{} },
		common.Uid: func() Handler { return &Uid{} },
	}

	go s.listen()
	go s.listenUpdates()
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
