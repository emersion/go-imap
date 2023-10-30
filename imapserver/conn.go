package imapserver

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const (
	cmdReadTimeout     = 30 * time.Second
	idleReadTimeout    = 35 * time.Minute // section 5.4 says 30min minimum
	literalReadTimeout = 5 * time.Minute

	respWriteTimeout    = 30 * time.Second
	literalWriteTimeout = 5 * time.Minute
)

var internalServerErrorResp = &imap.StatusResponse{
	Type: imap.StatusResponseTypeNo,
	Code: imap.ResponseCodeServerBug,
	Text: "Internal server error",
}

// A Conn represents an IMAP connection to the server.
type Conn struct {
	server   *Server
	br       *bufio.Reader
	bw       *bufio.Writer
	encMutex sync.Mutex

	mutex   sync.Mutex
	conn    net.Conn
	enabled imap.CapSet

	state   imap.ConnState
	session Session
}

func newConn(c net.Conn, server *Server) *Conn {
	rw := server.options.wrapReadWriter(c)
	br := bufio.NewReader(rw)
	bw := bufio.NewWriter(rw)
	return &Conn{
		conn:    c,
		server:  server,
		br:      br,
		bw:      bw,
		enabled: make(imap.CapSet),
	}
}

// NetConn returns the underlying connection that is wrapped by the IMAP
// connection.
//
// Writing to or reading from this connection directly will corrupt the IMAP
// session.
func (c *Conn) NetConn() net.Conn {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.conn
}

// Bye terminates the IMAP connection.
func (c *Conn) Bye(text string) error {
	respErr := c.writeStatusResp("", &imap.StatusResponse{
		Type: imap.StatusResponseTypeBye,
		Text: text,
	})
	closeErr := c.conn.Close()
	if respErr != nil {
		return respErr
	}
	return closeErr
}

func (c *Conn) serve() {
	defer func() {
		if v := recover(); v != nil {
			c.server.logger().Printf("panic handling command: %v\n%s", v, debug.Stack())
		}

		c.conn.Close()
	}()

	c.server.mutex.Lock()
	c.server.conns[c] = struct{}{}
	c.server.mutex.Unlock()
	defer func() {
		c.server.mutex.Lock()
		delete(c.server.conns, c)
		c.server.mutex.Unlock()
	}()

	var (
		greetingData *GreetingData
		err          error
	)
	c.session, greetingData, err = c.server.options.NewSession(c)
	if err != nil {
		var (
			resp    *imap.StatusResponse
			imapErr *imap.Error
		)
		if errors.As(err, &imapErr) && imapErr.Type == imap.StatusResponseTypeBye {
			resp = (*imap.StatusResponse)(imapErr)
		} else {
			c.server.logger().Printf("failed to create session: %v", err)
			resp = internalServerErrorResp
		}
		if err := c.writeStatusResp("", resp); err != nil {
			c.server.logger().Printf("failed to write greeting: %v", err)
		}
		return
	}

	defer func() {
		if c.session != nil {
			if err := c.session.Close(); err != nil {
				c.server.logger().Printf("failed to close session: %v", err)
			}
		}
	}()

	caps := c.server.options.caps()
	if _, ok := c.session.(SessionIMAP4rev2); !ok && caps.Has(imap.CapIMAP4rev2) {
		panic("imapserver: server advertises IMAP4rev2 but session doesn't support it")
	}
	if _, ok := c.session.(SessionNamespace); !ok && caps.Has(imap.CapNamespace) {
		panic("imapserver: server advertises NAMESPACE but session doesn't support it")
	}
	if _, ok := c.session.(SessionMove); !ok && caps.Has(imap.CapMove) {
		panic("imapserver: server advertises MOVE but session doesn't support it")
	}
	if _, ok := c.session.(SessionUnauthenticate); !ok && caps.Has(imap.CapUnauthenticate) {
		panic("imapserver: server advertises UNAUTHENTICATE but session doesn't support it")
	}

	c.state = imap.ConnStateNotAuthenticated
	statusType := imap.StatusResponseTypeOK
	if greetingData != nil && greetingData.PreAuth {
		c.state = imap.ConnStateAuthenticated
		statusType = imap.StatusResponseTypePreAuth
	}
	if err := c.writeCapabilityStatus("", statusType, "IMAP server ready"); err != nil {
		c.server.logger().Printf("failed to write greeting: %v", err)
		return
	}

	for {
		var readTimeout time.Duration
		switch c.state {
		case imap.ConnStateAuthenticated, imap.ConnStateSelected:
			readTimeout = idleReadTimeout
		default:
			readTimeout = cmdReadTimeout
		}
		c.setReadTimeout(readTimeout)

		dec := imapwire.NewDecoder(c.br, imapwire.ConnSideServer)
		dec.CheckBufferedLiteralFunc = c.checkBufferedLiteral

		if c.state == imap.ConnStateLogout || dec.EOF() {
			break
		}

		c.setReadTimeout(cmdReadTimeout)
		if err := c.readCommand(dec); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				c.server.logger().Printf("failed to read command: %v", err)
			}
			break
		}
	}
}

func (c *Conn) readCommand(dec *imapwire.Decoder) error {
	var tag, name string
	if !dec.ExpectAtom(&tag) || !dec.ExpectSP() || !dec.ExpectAtom(&name) {
		return fmt.Errorf("in command: %w", dec.Err())
	}
	name = strings.ToUpper(name)

	numKind := NumKindSeq
	if name == "UID" {
		numKind = NumKindUID
		var subName string
		if !dec.ExpectSP() || !dec.ExpectAtom(&subName) {
			return fmt.Errorf("in command: %w", dec.Err())
		}
		name = "UID " + strings.ToUpper(subName)
	}

	// TODO: handle multiple commands concurrently
	sendOK := true
	var err error
	switch name {
	case "NOOP", "CHECK":
		err = c.handleNoop(dec)
	case "LOGOUT":
		err = c.handleLogout(dec)
	case "CAPABILITY":
		err = c.handleCapability(dec)
	case "STARTTLS":
		err = c.handleStartTLS(tag, dec)
		sendOK = false
	case "AUTHENTICATE":
		err = c.handleAuthenticate(tag, dec)
		sendOK = false
	case "UNAUTHENTICATE":
		err = c.handleUnauthenticate(dec)
	case "LOGIN":
		err = c.handleLogin(tag, dec)
		sendOK = false
	case "ENABLE":
		err = c.handleEnable(dec)
	case "CREATE":
		err = c.handleCreate(dec)
	case "DELETE":
		err = c.handleDelete(dec)
	case "RENAME":
		err = c.handleRename(dec)
	case "SUBSCRIBE":
		err = c.handleSubscribe(dec)
	case "UNSUBSCRIBE":
		err = c.handleUnsubscribe(dec)
	case "STATUS":
		err = c.handleStatus(dec)
	case "LIST":
		err = c.handleList(dec)
	case "LSUB":
		err = c.handleLSub(dec)
	case "NAMESPACE":
		err = c.handleNamespace(dec)
	case "IDLE":
		err = c.handleIdle(dec)
	case "SELECT", "EXAMINE":
		err = c.handleSelect(tag, dec, name == "EXAMINE")
		sendOK = false
	case "CLOSE", "UNSELECT":
		err = c.handleUnselect(dec, name == "CLOSE")
	case "APPEND":
		err = c.handleAppend(tag, dec)
		sendOK = false
	case "FETCH", "UID FETCH":
		err = c.handleFetch(dec, numKind)
	case "EXPUNGE":
		err = c.handleExpunge(dec)
	case "UID EXPUNGE":
		err = c.handleUIDExpunge(dec)
	case "STORE", "UID STORE":
		err = c.handleStore(dec, numKind)
	case "COPY", "UID COPY":
		err = c.handleCopy(tag, dec, numKind)
		sendOK = false
	case "MOVE", "UID MOVE":
		err = c.handleMove(dec, numKind)
	case "SEARCH", "UID SEARCH":
		err = c.handleSearch(tag, dec, numKind)
	default:
		if c.state == imap.ConnStateNotAuthenticated {
			// Don't allow a single unknown command before authentication to
			// mitigate cross-protocol attacks:
			// https://www-archive.mozilla.org/projects/netlib/portbanning
			c.state = imap.ConnStateLogout
			defer c.Bye("Unknown command")
		}
		err = &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Unknown command",
		}
	}

	dec.DiscardLine()

	var (
		resp    *imap.StatusResponse
		imapErr *imap.Error
		decErr  *imapwire.DecoderExpectError
	)
	if errors.As(err, &imapErr) {
		resp = (*imap.StatusResponse)(imapErr)
	} else if errors.As(err, &decErr) {
		resp = &imap.StatusResponse{
			Type: imap.StatusResponseTypeBad,
			Code: imap.ResponseCodeClientBug,
			Text: "Syntax error: " + decErr.Message,
		}
	} else if err != nil {
		c.server.logger().Printf("handling %v command: %v", name, err)
		resp = internalServerErrorResp
	} else {
		if !sendOK {
			return nil
		}
		if err := c.poll(name); err != nil {
			return err
		}
		resp = &imap.StatusResponse{
			Type: imap.StatusResponseTypeOK,
			Text: fmt.Sprintf("%v completed", name),
		}
	}
	return c.writeStatusResp(tag, resp)
}

func (c *Conn) handleNoop(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}
	return nil
}

func (c *Conn) handleLogout(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	c.state = imap.ConnStateLogout

	return c.writeStatusResp("", &imap.StatusResponse{
		Type: imap.StatusResponseTypeBye,
		Text: "Logging out",
	})
}

func (c *Conn) handleDelete(dec *imapwire.Decoder) error {
	var name string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&name) || !dec.ExpectCRLF() {
		return dec.Err()
	}
	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}
	return c.session.Delete(name)
}

func (c *Conn) handleRename(dec *imapwire.Decoder) error {
	var oldName, newName string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&oldName) || !dec.ExpectSP() || !dec.ExpectMailbox(&newName) || !dec.ExpectCRLF() {
		return dec.Err()
	}
	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}
	return c.session.Rename(oldName, newName)
}

func (c *Conn) handleSubscribe(dec *imapwire.Decoder) error {
	var name string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&name) || !dec.ExpectCRLF() {
		return dec.Err()
	}
	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}
	return c.session.Subscribe(name)
}

func (c *Conn) handleUnsubscribe(dec *imapwire.Decoder) error {
	var name string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&name) || !dec.ExpectCRLF() {
		return dec.Err()
	}
	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}
	return c.session.Unsubscribe(name)
}

func (c *Conn) checkBufferedLiteral(size int64, nonSync bool) error {
	if size > 4096 {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeTooBig,
			Text: "Literals are limited to 4096 bytes for this command",
		}
	}

	return c.acceptLiteral(size, nonSync)
}

func (c *Conn) acceptLiteral(size int64, nonSync bool) error {
	if nonSync && size > 4096 && !c.server.options.caps().Has(imap.CapLiteralPlus) {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Non-synchronizing literals are limited to 4096 bytes",
		}
	}

	if nonSync {
		return nil
	}

	return c.writeContReq("Ready for literal data")
}

func (c *Conn) canAuth() bool {
	if c.state != imap.ConnStateNotAuthenticated {
		return false
	}
	_, isTLS := c.conn.(*tls.Conn)
	return isTLS || c.server.options.InsecureAuth
}

func (c *Conn) writeStatusResp(tag string, statusResp *imap.StatusResponse) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return writeStatusResp(enc.Encoder, tag, statusResp)
}

func (c *Conn) writeContReq(text string) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return writeContReq(enc.Encoder, text)
}

func (c *Conn) writeCapabilityStatus(tag string, typ imap.StatusResponseType, text string) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return writeCapabilityStatus(enc.Encoder, tag, typ, c.availableCaps(), text)
}

func (c *Conn) checkState(state imap.ConnState) error {
	if state == imap.ConnStateAuthenticated && c.state == imap.ConnStateSelected {
		return nil
	}
	if c.state != state {
		return newClientBugError(fmt.Sprintf("This command is only valid in the %s state", state))
	}
	return nil
}

func (c *Conn) setReadTimeout(dur time.Duration) {
	if dur > 0 {
		c.conn.SetReadDeadline(time.Now().Add(dur))
	} else {
		c.conn.SetReadDeadline(time.Time{})
	}
}

func (c *Conn) setWriteTimeout(dur time.Duration) {
	if dur > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(dur))
	} else {
		c.conn.SetWriteDeadline(time.Time{})
	}
}

func (c *Conn) poll(cmd string) error {
	switch c.state {
	case imap.ConnStateAuthenticated, imap.ConnStateSelected:
		// nothing to do
	default:
		return nil
	}

	allowExpunge := true
	switch cmd {
	case "FETCH", "STORE", "SEARCH":
		allowExpunge = false
	}

	w := &UpdateWriter{conn: c, allowExpunge: allowExpunge}
	return c.session.Poll(w, allowExpunge)
}

type responseEncoder struct {
	*imapwire.Encoder
	conn *Conn
}

func newResponseEncoder(conn *Conn) *responseEncoder {
	conn.mutex.Lock()
	quotedUTF8 := conn.enabled.Has(imap.CapIMAP4rev2)
	conn.mutex.Unlock()

	wireEnc := imapwire.NewEncoder(conn.bw, imapwire.ConnSideServer)
	wireEnc.QuotedUTF8 = quotedUTF8

	conn.encMutex.Lock() // released by responseEncoder.end
	conn.setWriteTimeout(respWriteTimeout)
	return &responseEncoder{
		Encoder: wireEnc,
		conn:    conn,
	}
}

func (enc *responseEncoder) end() {
	if enc.Encoder == nil {
		panic("imapserver: responseEncoder.end called twice")
	}
	enc.Encoder = nil
	enc.conn.setWriteTimeout(0)
	enc.conn.encMutex.Unlock()
}

func (enc *responseEncoder) Literal(size int64) io.WriteCloser {
	enc.conn.setWriteTimeout(literalWriteTimeout)
	return literalWriter{
		WriteCloser: enc.Encoder.Literal(size, nil),
		conn:        enc.conn,
	}
}

type literalWriter struct {
	io.WriteCloser
	conn *Conn
}

func (lw literalWriter) Close() error {
	lw.conn.setWriteTimeout(respWriteTimeout)
	return lw.WriteCloser.Close()
}

func writeStatusResp(enc *imapwire.Encoder, tag string, statusResp *imap.StatusResponse) error {
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

func writeCapabilityOK(enc *imapwire.Encoder, tag string, caps []imap.Cap, text string) error {
	return writeCapabilityStatus(enc, tag, imap.StatusResponseTypeOK, caps, text)
}

func writeCapabilityStatus(enc *imapwire.Encoder, tag string, typ imap.StatusResponseType, caps []imap.Cap, text string) error {
	if tag == "" {
		tag = "*"
	}

	enc.Atom(tag).SP().Atom(string(typ)).SP().Special('[').Atom("CAPABILITY")
	for _, c := range caps {
		enc.SP().Atom(string(c))
	}
	enc.Special(']').SP().Text(text)
	return enc.CRLF()
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

// UpdateWriter writes status updates.
type UpdateWriter struct {
	conn         *Conn
	allowExpunge bool
}

// WriteExpunge writes an EXPUNGE response.
func (w *UpdateWriter) WriteExpunge(seqNum uint32) error {
	if !w.allowExpunge {
		return fmt.Errorf("imapserver: EXPUNGE updates are not allowed in this context")
	}
	return w.conn.writeExpunge(seqNum)
}

// WriteNumMessages writes an EXISTS response.
func (w *UpdateWriter) WriteNumMessages(n uint32) error {
	return w.conn.writeExists(n)
}

// WriteMailboxFlags writes a FLAGS response.
func (w *UpdateWriter) WriteMailboxFlags(flags []imap.Flag) error {
	return w.conn.writeFlags(flags)
}

// WriteMessageFlags writes a FETCH response with FLAGS.
func (w *UpdateWriter) WriteMessageFlags(seqNum uint32, uid imap.UID, flags []imap.Flag) error {
	fetchWriter := &FetchWriter{conn: w.conn}
	respWriter := fetchWriter.CreateMessage(seqNum)
	if uid != 0 {
		respWriter.WriteUID(uid)
	}
	respWriter.WriteFlags(flags)
	return respWriter.Close()
}
