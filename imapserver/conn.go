package imapserver

import (
	"bufio"
	"fmt"
	"net"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

type conn struct {
	conn     net.Conn
	server   *Server
	br       *bufio.Reader
	bw       *bufio.Writer
	dec      *imapwire.Decoder
	encMutex sync.Mutex

	state imap.ConnState
}

func newConn(c net.Conn, server *Server) *conn {
	br := bufio.NewReader(c)
	bw := bufio.NewWriter(c)
	return &conn{
		conn:   c,
		server: server,
		br:     br,
		bw:     bw,
		dec:    imapwire.NewDecoder(br),
	}
}

func (c *conn) serve() {
	defer func() {
		if v := recover(); v != nil {
			c.server.Logger.Printf("panic handling command: %v\n%s", v, debug.Stack())
		}

		c.conn.Close()
	}()

	c.state = imap.ConnStateNotAuthenticated
	err := c.writeStatusResp("", &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Text: "IMAP4rev2 server ready",
	})
	if err != nil {
		c.server.Logger.Printf("failed to write greeting: %v", err)
		return
	}

	for {
		if c.state == imap.ConnStateLogout || c.dec.EOF() {
			break
		}

		if err := c.readCommand(); err != nil {
			c.server.Logger.Printf("failed to read command: %v", err)
			break
		}
	}
}

func (c *conn) readCommand() error {
	var tag, name string
	if !c.dec.ExpectAtom(&tag) || !c.dec.ExpectSP() || !c.dec.ExpectAtom(&name) {
		return fmt.Errorf("in command: %v", c.dec.Err())
	}
	name = strings.ToUpper(name)

	// TODO: handle multiple commands concurrently
	var err error
	switch name {
	case "NOOP":
		// do nothing
	case "LOGOUT":
		err = c.handleLogout()
	case "CAPABILITY":
		err = c.handleCapability()
	default:
		var text string
		c.dec.Text(&text)

		err = &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown command",
		}
	}
	if !c.dec.CRLF() {
		return c.dec.Err()
	}

	var resp *imap.StatusResponse
	if imapErr, ok := err.(*imap.Error); ok {
		resp = (*imap.StatusResponse)(imapErr)
	} else if err != nil {
		c.server.Logger.Printf("handling %v command: %v", name, err)
		resp = &imap.StatusResponse{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeServerBug,
			Text: "Internal server error",
		}
	} else {
		resp = &imap.StatusResponse{
			Type: imap.StatusResponseTypeOK,
			Text: fmt.Sprintf("%v completed", name),
		}
	}
	return c.writeStatusResp(tag, resp)
}

func (c *conn) handleLogout() error {
	c.state = imap.ConnStateLogout
	return c.writeStatusResp("", &imap.StatusResponse{
		Type: imap.StatusResponseTypeBye,
		Text: "Logging out",
	})
}

func (c *conn) handleCapability() error {
	enc := newResponseEncoder(c)
	enc.Atom("*").SP().Atom("CAPABILITY").SP().Atom(string(imap.CapIMAP4rev2))
	return enc.close()
}

func (c *conn) writeStatusResp(tag string, statusResp *imap.StatusResponse) error {
	enc := newResponseEncoder(c)

	if tag == "" {
		tag = "*"
	}
	enc.Atom(tag).SP().Atom(string(statusResp.Type)).SP()
	if statusResp.Code != "" {
		enc.Atom(fmt.Sprintf("[%v]", statusResp.Code)).SP()
	}
	enc.Text(statusResp.Text)

	return enc.close()
}

type responseEncoder struct {
	*imapwire.Encoder
	conn *conn
}

func newResponseEncoder(conn *conn) *responseEncoder {
	conn.encMutex.Lock() // released by responseEncoder.close
	return &responseEncoder{
		Encoder: imapwire.NewEncoder(conn.bw),
		conn:    conn,
	}
}

func (enc *responseEncoder) close() error {
	err := enc.CRLF()
	enc.Encoder = nil
	enc.conn.encMutex.Unlock()
	return err
}
