package imapserver_test

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// parseWire is a raw connection to a server: these cases are about the exact
// status line, which a client library hides.
type parseWire struct {
	t    *testing.T
	conn net.Conn
	rd   *bufio.Reader
}

func dialParseServer(t *testing.T, newSession func(imapserver.Session) imapserver.Session) *parseWire {
	t.Helper()
	mem := imapmemserver.New()
	u := imapmemserver.NewUser("u", "p")
	if err := u.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	mem.AddUser(u)
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			s := imapserver.Session(mem.NewSession())
			if newSession != nil {
				s = newSession(s)
			}
			return s, nil, nil
		},
		InsecureAuth: true,
		Caps:         imap.CapSet{imap.CapIMAP4rev1: struct{}{}},
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go srv.Serve(ln) //nolint:errcheck

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	w := &parseWire{t: t, conn: c, rd: bufio.NewReader(c)}
	w.line()
	fmt.Fprint(w.conn, "a1 LOGIN u p\r\n")
	w.until("a1")
	return w
}

func (w *parseWire) line() string {
	w.t.Helper()
	_ = w.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	l, err := w.rd.ReadString('\n')
	if err != nil {
		w.t.Fatalf("read: %v", err)
	}
	return strings.TrimRight(l, "\r\n")
}

// until returns the tagged status line for tag.
func (w *parseWire) until(tag string) string {
	w.t.Helper()
	for {
		l := w.line()
		if strings.HasPrefix(l, tag+" ") {
			return l
		}
	}
}

// A command the grammar refuses is answered BAD, not NO [SERVERBUG]: a parse
// failure arriving as a plain error tells the client the server is broken.
func TestMalformedCommandIsBad(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  string
	}{
		{"sequence-set", `a2 SEARCH HEADER Subject fileinto test`},
		{"mailbox name", `a2 SELECT "&"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := dialParseServer(t, nil)
			fmt.Fprint(w.conn, tc.cmd+"\r\n")
			got := w.until("a2")
			if strings.Contains(got, "SERVERBUG") {
				t.Errorf("malformed command answered %q; want BAD", got)
			}
			if !strings.HasPrefix(got, "a2 BAD") {
				t.Errorf("answer was %q; want BAD", got)
			}
		})
	}
}

// failingSelect answers a well-formed SELECT with a plain error, which is what a
// server fault looks like to the connection loop.
type failingSelect struct {
	imapserver.Session
}

func (s *failingSelect) Select(string, *imap.SelectOptions) (*imap.SelectData, error) {
	return nil, errors.New("boom")
}

// A plain error from the handler is still NO [SERVERBUG]: this classifies parse
// failures, it does not answer BAD to everything.
func TestHandlerErrorIsServerBug(t *testing.T) {
	w := dialParseServer(t, func(s imapserver.Session) imapserver.Session {
		return &failingSelect{Session: s}
	})
	fmt.Fprint(w.conn, "a2 SELECT INBOX\r\n")
	if got := w.until("a2"); !strings.Contains(got, "SERVERBUG") {
		t.Errorf("handler error answered %q; want NO [SERVERBUG]", got)
	}
}
