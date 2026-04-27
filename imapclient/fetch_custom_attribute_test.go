package imapclient_test

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// scriptedServer drives a minimal IMAP conversation over a net.Pipe so the
// client parser can be exercised against hand-crafted server responses.
// Unhandled commands receive a BAD response so unexpected client behavior
// surfaces as a protocol error rather than a hang.
type scriptedServer struct {
	conn   net.Conn
	br     *bufio.Reader
	bw     *bufio.Writer
	mu     sync.Mutex
	closed bool
}

func newScriptedServer(_ *testing.T, conn net.Conn) *scriptedServer {
	return &scriptedServer{
		conn: conn,
		br:   bufio.NewReader(conn),
		bw:   bufio.NewWriter(conn),
	}
}

func (s *scriptedServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.conn.Close()
}

func (s *scriptedServer) writeLine(line string) error {
	if _, err := s.bw.WriteString(line + "\r\n"); err != nil {
		return err
	}
	return s.bw.Flush()
}

func (s *scriptedServer) readLine() (string, error) {
	line, err := s.br.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// expectCommand reads a client command line and splits off the tag.
func (s *scriptedServer) expectCommand() (tag, body string, err error) {
	line, err := s.readLine()
	if err != nil {
		return "", "", err
	}
	idx := strings.IndexByte(line, ' ')
	if idx < 0 {
		return "", "", fmt.Errorf("malformed command line: %q", line)
	}
	return line[:idx], line[idx+1:], nil
}

// driveLoginAndSelect handles the greeting/CAPABILITY/LOGIN/SELECT preamble
// and returns once the mailbox is selected.
func (s *scriptedServer) driveLoginAndSelect() error {
	if err := s.writeLine("* OK [CAPABILITY IMAP4rev1 IMAP4rev2 X-GM-EXT-1] Server ready"); err != nil {
		return err
	}

	for {
		tag, body, err := s.expectCommand()
		if err != nil {
			return err
		}
		upper := strings.ToUpper(body)
		switch {
		case strings.HasPrefix(upper, "CAPABILITY"):
			if err := s.writeLines(
				"* CAPABILITY IMAP4rev1 IMAP4rev2 X-GM-EXT-1",
				tag+" OK CAPABILITY completed",
			); err != nil {
				return err
			}
		case strings.HasPrefix(upper, "LOGIN"):
			if err := s.writeLine(tag + " OK LOGIN completed"); err != nil {
				return err
			}
		case strings.HasPrefix(upper, "SELECT"):
			return s.writeLines(
				"* 1 EXISTS",
				"* 0 RECENT",
				"* FLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft)",
				"* OK [PERMANENTFLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft)] Permanent flags",
				"* OK [UIDNEXT 2] Predicted next UID",
				"* OK [UIDVALIDITY 1] UIDs valid",
				tag+" OK [READ-WRITE] SELECT completed",
			)
		default:
			if err := s.writeLine(tag + " BAD unexpected command in preamble: " + body); err != nil {
				return err
			}
		}
	}
}

func (s *scriptedServer) writeLines(lines ...string) error {
	for _, line := range lines {
		if err := s.writeLine(line); err != nil {
			return err
		}
	}
	return nil
}

func TestFetch_CustomAttributes_Parsing(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	_ = clientConn.SetDeadline(time.Now().Add(10 * time.Second))
	_ = serverConn.SetDeadline(time.Now().Add(10 * time.Second))

	number64Decoder := func(d *imapclient.AttributeDecoder) (imapclient.CustomAttribute, error) {
		var n int64
		if !d.ExpectNumber64(&n) {
			return imapclient.CustomAttribute{}, d.Err()
		}
		return imapclient.NumberAttribute(uint64(n)), nil
	}
	options := &imapclient.Options{
		CustomAttributeDecoders: map[string]imapclient.CustomAttributeDecoderFunc{
			"X-GM-MSGID": number64Decoder,
			"X-GM-THRID": number64Decoder,
			"X-GM-LABELS": func(d *imapclient.AttributeDecoder) (imapclient.CustomAttribute, error) {
				var labels []string
				err := d.ExpectList(func() error {
					prefix := ""
					if d.Special('\\') {
						prefix = `\`
					}
					var label string
					if !d.String(&label) && !d.Atom(&label) {
						return errors.New("expected label atom or string")
					}
					labels = append(labels, prefix+label)
					return nil
				})
				if err != nil {
					return imapclient.CustomAttribute{}, err
				}
				return imapclient.StringListAttribute(labels), nil
			},
		},
	}
	client := imapclient.New(clientConn, options)
	defer client.Close()

	server := newScriptedServer(t, serverConn)
	defer server.Close()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- func() error {
			if err := server.driveLoginAndSelect(); err != nil {
				return fmt.Errorf("preamble: %w", err)
			}

			tag, body, err := server.expectCommand()
			if err != nil {
				return fmt.Errorf("read fetch: %w", err)
			}
			upper := strings.ToUpper(body)
			if !strings.HasPrefix(upper, "UID FETCH") && !strings.HasPrefix(upper, "FETCH") {
				return fmt.Errorf("unexpected command: %q", body)
			}
			for _, want := range []string{"X-GM-MSGID", "X-GM-THRID", "X-GM-LABELS"} {
				if !strings.Contains(upper, want) {
					return fmt.Errorf("FETCH command %q missing %q", body, want)
				}
			}

			if err := server.writeLines(
				`* 1 FETCH (UID 100 X-GM-MSGID 1544546839788733155 X-GM-THRID 1234567890123456789 X-GM-LABELS (\Inbox \Important "Foo/Bar"))`,
				tag+" OK FETCH completed",
			); err != nil {
				return err
			}

			tag, body, err = server.expectCommand()
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
					return nil
				}
				return err
			}
			if strings.HasPrefix(strings.ToUpper(body), "LOGOUT") {
				_ = server.writeLines("* BYE Logging out", tag+" OK LOGOUT completed")
			}
			return nil
		}()
	}()

	if err := client.Login("u", "p").Wait(); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select: %v", err)
	}

	fetchOptions := &imap.FetchOptions{
		UID:              true,
		CustomAttributes: []string{"X-GM-MSGID", "X-GM-THRID", "X-GM-LABELS"},
	}
	messages, err := client.Fetch(imap.UIDSetNum(100), fetchOptions).Collect()
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(messages))
	}

	msg := messages[0]
	if msg.UID != imap.UID(100) {
		t.Errorf("UID = %d, want 100", msg.UID)
	}

	gotMsgID, ok := msg.FindCustomAttribute("X-GM-MSGID")
	if !ok {
		t.Fatalf("FindCustomAttribute(X-GM-MSGID) missing; have %v", msg.CustomAttributes)
	}
	msgID, ok := gotMsgID.GetNumber()
	if !ok {
		t.Fatalf("X-GM-MSGID is not a Number; value = %+v", gotMsgID)
	}
	if msgID != 1544546839788733155 {
		t.Errorf("X-GM-MSGID = %d, want 1544546839788733155", msgID)
	}

	gotThrID, ok := msg.FindCustomAttribute("x-gm-thrid")
	if !ok {
		t.Fatalf("FindCustomAttribute(x-gm-thrid) missing")
	}
	thrID, ok := gotThrID.GetNumber()
	if !ok {
		t.Fatalf("X-GM-THRID is not a Number; value = %+v", gotThrID)
	}
	if thrID != 1234567890123456789 {
		t.Errorf("X-GM-THRID = %d, want 1234567890123456789", thrID)
	}

	gotLabels, ok := msg.FindCustomAttribute("X-GM-LABELS")
	if !ok {
		t.Fatalf("FindCustomAttribute(X-GM-LABELS) missing")
	}
	labels, ok := gotLabels.GetStringList()
	if !ok {
		t.Fatalf("X-GM-LABELS is not a StringList; value = %+v", gotLabels)
	}
	wantLabels := []string{`\Inbox`, `\Important`, "Foo/Bar"}
	if len(labels) != len(wantLabels) {
		t.Fatalf("len(labels) = %d (%v), want %d (%v)", len(labels), labels, len(wantLabels), wantLabels)
	}
	for i, want := range wantLabels {
		if labels[i] != want {
			t.Errorf("labels[%d] = %q, want %q", i, labels[i], want)
		}
	}

	if err := client.Logout().Wait(); err != nil {
		if !errors.Is(err, io.ErrClosedPipe) && !errors.Is(err, io.EOF) {
			t.Logf("Logout: %v", err)
		}
	}

	server.Close()
	<-serverDone
}

func TestFetch_CustomAttributes_UnregisteredAttributeStillFails(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	deadline := time.Now().Add(10 * time.Second)
	_ = clientConn.SetDeadline(deadline)
	_ = serverConn.SetDeadline(deadline)

	client := imapclient.New(clientConn, nil)
	defer client.Close()

	server := newScriptedServer(t, serverConn)
	defer server.Close()

	go func() {
		if err := server.driveLoginAndSelect(); err != nil {
			return
		}
		tag, _, err := server.expectCommand()
		if err != nil {
			return
		}
		_ = server.writeLines(
			`* 1 FETCH (UID 100 X-GM-MSGID 1544546839788733155)`,
			tag+" OK FETCH completed",
		)
	}()

	if err := client.Login("u", "p").Wait(); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select: %v", err)
	}

	_, err := client.Fetch(imap.UIDSetNum(100), &imap.FetchOptions{UID: true}).Collect()
	if err == nil {
		t.Fatalf("Fetch must fail for unsupported atom when no decoder is registered")
	}
	if !strings.Contains(err.Error(), "unsupported msg-att name") {
		t.Errorf("unexpected error: %v", err)
	}
}
