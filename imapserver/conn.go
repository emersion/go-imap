package imapserver

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

var internalServerErrorResp = &imap.StatusResponse{
	Type: imap.StatusResponseTypeNo,
	Code: imap.ResponseCodeServerBug,
	Text: "Internal server error",
}

type conn struct {
	conn     net.Conn
	server   *Server
	br       *bufio.Reader
	bw       *bufio.Writer
	encMutex sync.Mutex

	state   imap.ConnState
	session Session
}

func newConn(c net.Conn, server *Server) *conn {
	br := bufio.NewReader(c)
	bw := bufio.NewWriter(c)
	return &conn{
		conn:   c,
		server: server,
		br:     br,
		bw:     bw,
	}
}

func (c *conn) serve() {
	defer func() {
		if v := recover(); v != nil {
			c.server.Logger.Printf("panic handling command: %v\n%s", v, debug.Stack())
		}

		c.conn.Close()
	}()

	var err error
	c.session, err = c.server.NewSession()
	if err != nil {
		var (
			resp    *imap.StatusResponse
			imapErr *imap.Error
		)
		if errors.As(err, &imapErr) {
			resp = (*imap.StatusResponse)(imapErr)
		} else {
			c.server.Logger.Printf("failed to create session: %v", err)
			resp = internalServerErrorResp
		}
		if err := c.writeStatusResp("", resp); err != nil {
			c.server.Logger.Printf("failed to write greeting: %v", err)
		}
		return
	}

	defer func() {
		if c.session != nil {
			if err := c.session.Close(); err != nil {
				c.server.Logger.Printf("failed to close session: %v", err)
			}
		}
	}()

	c.state = imap.ConnStateNotAuthenticated
	err = c.writeStatusResp("", &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Text: "IMAP4rev2 server ready",
	})
	if err != nil {
		c.server.Logger.Printf("failed to write greeting: %v", err)
		return
	}

	for {
		dec := imapwire.NewDecoder(c.br)
		if c.state == imap.ConnStateLogout || dec.EOF() {
			break
		}

		if err := c.readCommand(dec); err != nil {
			c.server.Logger.Printf("failed to read command: %v", err)
			break
		}
	}
}

func (c *conn) readCommand(dec *imapwire.Decoder) error {
	var tag, name string
	if !dec.ExpectAtom(&tag) || !dec.ExpectSP() || !dec.ExpectAtom(&name) {
		return fmt.Errorf("in command: %w", dec.Err())
	}
	name = strings.ToUpper(name)

	// TODO: handle multiple commands concurrently
	var err error
	switch name {
	case "NOOP":
		err = c.handleNoop(dec)
	case "LOGOUT":
		err = c.handleLogout(dec)
	case "CAPABILITY":
		err = c.handleCapability(dec)
	case "AUTHENTICATE":
		err = c.handleAuthenticate(dec)
	case "LOGIN":
		err = c.handleLogin(dec)
	case "STATUS":
		err = c.handleStatus(dec)
	case "LIST":
		err = c.handleList(dec)
	case "IDLE":
		err = c.handleIdle(dec)
	case "SELECT", "EXAMINE":
		err = c.handleSelect(dec, name == "EXAMINE")
	case "CLOSE", "UNSELECT":
		err = c.handleUnselect(dec, name == "CLOSE")
	case "APPEND":
		err = c.handleAppend(dec)
	case "FETCH":
		err = c.handleFetch(dec)
	default:
		discardLine(dec)
		err = &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown command",
		}
	}

	var (
		resp   *imap.StatusResponse
		decErr *imapwire.DecoderExpectError
	)
	if imapErr, ok := err.(*imap.Error); ok {
		resp = (*imap.StatusResponse)(imapErr)
	} else if errors.As(err, &decErr) {
		discardLine(dec)
		resp = &imap.StatusResponse{
			Type: imap.StatusResponseTypeBad,
			Code: imap.ResponseCodeClientBug,
			Text: "Syntax error: " + decErr.Message,
		}
	} else if err != nil {
		c.server.Logger.Printf("handling %v command: %v", name, err)
		resp = internalServerErrorResp
	} else {
		resp = &imap.StatusResponse{
			Type: imap.StatusResponseTypeOK,
			Text: fmt.Sprintf("%v completed", name),
		}
	}
	return c.writeStatusResp(tag, resp)
}

func (c *conn) handleNoop(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}
	return nil
}

func (c *conn) handleLogout(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	c.state = imap.ConnStateLogout

	return c.writeStatusResp("", &imap.StatusResponse{
		Type: imap.StatusResponseTypeBye,
		Text: "Logging out",
	})
}

func (c *conn) handleCapability(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("CAPABILITY").SP().Atom(string(imap.CapIMAP4rev2))
	return enc.CRLF()
}

func (c *conn) handleLogin(dec *imapwire.Decoder) error {
	var username, password string
	if !dec.ExpectSP() || !dec.ExpectAString(&username) || !dec.ExpectSP() || !dec.ExpectAString(&password) || !dec.ExpectCRLF() {
		return dec.Err()
	}
	if err := c.checkState(imap.ConnStateNotAuthenticated); err != nil {
		return err
	}
	if err := c.session.Login(username, password); err != nil {
		return err
	}
	c.state = imap.ConnStateAuthenticated
	return nil
}

func (c *conn) handleIdle(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	enc := newResponseEncoder(c)
	defer enc.end()

	if err := writeContReq(enc.Encoder, "idling"); err != nil {
		return err
	}

	line, isPrefix, err := c.br.ReadLine()
	if err != nil {
		return err
	} else if isPrefix || string(line) != "DONE" {
		return newClientBugError("Syntax error: expected DONE to end IDLE command")
	}

	return nil
}

func (c *conn) writeStatusResp(tag string, statusResp *imap.StatusResponse) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	if tag == "" {
		tag = "*"
	}
	enc.Atom(tag).SP().Atom(string(statusResp.Type)).SP()
	if statusResp.Code != "" {
		enc.Atom(fmt.Sprintf("[%v]", statusResp.Code)).SP()
	}
	enc.Text(statusResp.Text)
	return enc.CRLF()
}

func (c *conn) checkState(state imap.ConnState) error {
	if state == imap.ConnStateAuthenticated && c.state == imap.ConnStateSelected {
		return nil
	}
	if c.state != state {
		return newClientBugError(fmt.Sprintf("This command is only valid in the %s state", state))
	}
	return nil
}

type responseEncoder struct {
	*imapwire.Encoder
	conn *conn
}

func newResponseEncoder(conn *conn) *responseEncoder {
	conn.encMutex.Lock() // released by responseEncoder.end
	return &responseEncoder{
		Encoder: imapwire.NewEncoder(conn.bw),
		conn:    conn,
	}
}

func (enc *responseEncoder) end() {
	if enc.Encoder == nil {
		panic("imapserver: responseEncoder.end called twice")
	}
	enc.Encoder = nil
	enc.conn.encMutex.Unlock()
}

func discardLine(dec *imapwire.Decoder) {
	var text string
	dec.Text(&text)
	dec.CRLF()
}

func writeContReq(enc *imapwire.Encoder, text string) error {
	return enc.Atom("+").SP().Text(text).CRLF()
}

func newClientBugError(text string) error {
	return &imap.Error{
		Type: imap.StatusResponseTypeBad,
		Code: imap.ResponseCodeClientBug,
		Text: text,
	}
}
