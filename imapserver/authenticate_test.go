package imapserver_test

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-sasl"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// emptyStepServer is a SASL mechanism whose only challenge is empty and which
// expects an empty response to it.
type emptyStepServer struct {
	step int
}

func (s *emptyStepServer) Next(resp []byte) (challenge []byte, done bool, err error) {
	s.step++
	switch s.step {
	case 1:
		return []byte{}, false, nil
	case 2:
		if len(resp) != 0 {
			return nil, false, sasl.ErrUnexpectedClientResponse
		}
		return nil, true, nil
	default:
		return nil, false, sasl.ErrUnexpectedClientResponse
	}
}

type saslSession struct {
	imapserver.Session
}

func (s *saslSession) AuthenticateMechanisms() []string {
	return []string{"X-EMPTY-STEP"}
}

func (s *saslSession) Authenticate(mech string) (sasl.Server, error) {
	return &emptyStepServer{}, nil
}

func TestAuthenticate_emptyContinuation(t *testing.T) {
	for _, tc := range []struct {
		name string
		resp string
	}{
		{"empty line", ""},
		// Sent by older versions of imapclient.
		{"zero-length marker", "="},
	} {
		t.Run(tc.name, func(t *testing.T) {
			memServer := imapmemserver.New()
			server := imapserver.New(&imapserver.Options{
				NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
					return &saslSession{memServer.NewSession()}, nil, nil
				},
				InsecureAuth: true,
				Caps:         imap.CapSet{imap.CapIMAP4rev1: {}},
			})
			ln, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				t.Fatalf("net.Listen() = %v", err)
			}
			defer server.Close()
			go server.Serve(ln)

			conn, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatalf("net.Dial() = %v", err)
			}
			defer conn.Close()
			br := bufio.NewReader(conn)

			readLine := func() string {
				t.Helper()
				line, err := br.ReadString('\n')
				if err != nil {
					t.Fatalf("ReadString() = %v", err)
				}
				return strings.TrimRight(line, "\r\n")
			}

			readLine() // greeting
			if _, err := conn.Write([]byte("a AUTHENTICATE X-EMPTY-STEP\r\n")); err != nil {
				t.Fatalf("Write() = %v", err)
			}
			if line := readLine(); line != "+ " {
				t.Fatalf("continuation request = %q, want %q", line, "+ ")
			}
			if _, err := conn.Write([]byte(tc.resp + "\r\n")); err != nil {
				t.Fatalf("Write() = %v", err)
			}
			if line := readLine(); !strings.HasPrefix(line, "a OK") {
				t.Fatalf("status response = %q, want a OK", line)
			}
		})
	}
}
