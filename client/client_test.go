package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap"
)

type cmdScanner struct {
	scanner *bufio.Scanner
}

func (s *cmdScanner) ScanLine() string {
	s.scanner.Scan()
	return s.scanner.Text()
}

// Deprecated
func (s *cmdScanner) Scan() (tag string, cmd string) {
	parts := strings.SplitN(s.ScanLine(), " ", 2)
	return parts[0], parts[1]
}

func (s *cmdScanner) ScanCmd() (tag string, cmd string) {
	parts := strings.SplitN(s.ScanLine(), " ", 2)
	return parts[0], parts[1]
}

func newCmdScanner(r io.Reader) *cmdScanner {
	return &cmdScanner{
		scanner: bufio.NewScanner(r),
	}
}

type serverConn struct {
	*cmdScanner
	io.WriteCloser
}

func (c *serverConn) WriteString(s string) (n int, err error) {
	return io.WriteString(c.WriteCloser, s)
}

func newTestClient(t *testing.T) (c *Client, s *serverConn) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		greeting := "* OK [CAPABILITY IMAP4rev1 STARTTLS AUTH=PLAIN] Server ready.\r\n"
		if _, err := io.WriteString(conn, greeting); err != nil {
			panic(err)
		}

		s = &serverConn{newCmdScanner(conn), conn}
		close(done)
	}()

	c, err = Dial(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	<-done
	return
}

type ClientTester func(c *Client) error
type ServerTester func(c net.Conn)

// Deprecated
func testClient(t *testing.T, ct ClientTester, st ServerTester) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	done := make(chan error)
	go (func() {
		c, err := Dial(l.Addr().String())
		if err != nil {
			done <- err
			return
		}

		err = ct(c)
		if err != nil {
			fmt.Println("Client error:", err)
			done <- err
			return
		}

		c.state = imap.LogoutState
		done <- nil
	})()

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	greeting := "* OK [CAPABILITY IMAP4rev1 STARTTLS AUTH=PLAIN] Server ready.\r\n"
	if _, err = io.WriteString(conn, greeting); err != nil {
		t.Fatal(err)
	}

	st(conn)

	err = <-done
	if err != nil {
		t.Fatal(err)
	}

	conn.Close()
}

func TestClient(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	if ok, err := c.Support("IMAP4rev1"); err != nil {
		t.Fatal("c.Support(IMAP4rev1) =", err)
	} else if !ok {
		t.Fatal("c.Support(IMAP4rev1) = false, want true")
	}
}

func TestClient_SetDebug(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	var b bytes.Buffer
	c.SetDebug(&b)

	done := make(chan error)
	go func() {
		_, err := c.Capability()
		done <- err
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "CAPABILITY" {
		t.Fatal("Bad command:", cmd)
	}

	s.WriteString("* CAPABILITY IMAP4rev1\r\n")
	s.WriteString(tag+" OK CAPABILITY completed.\r\n")

	if err := <-done; err != nil {
		t.Fatal("c.Capability() =", err)
	}

	if b.Len() == 0 {
		t.Error("empty debug buffer")
	}
}

func TestClient_unilateral(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	c.locker.Lock()
	c.state = imap.SelectedState
	c.mailbox = imap.NewMailboxStatus("INBOX", nil)
	c.locker.Unlock()

	statuses := make(chan *imap.MailboxStatus, 1)
	c.MailboxUpdates = statuses
	expunges := make(chan uint32, 1)
	c.Expunges = expunges
	messages := make(chan *imap.Message, 1)
	c.MessageUpdates = messages
	infos := make(chan *imap.StatusResp, 1)
	c.Infos = infos
	warns := make(chan *imap.StatusResp, 1)
	c.Warnings = warns
	errors := make(chan *imap.StatusResp, 1)
	c.Errors = errors

	s.WriteString("* 42 EXISTS\r\n")
	if status := <-statuses; status.Messages != 42 {
		t.Errorf("Invalid messages count: expected %v but got %v", 42, status.Messages)
	}

	s.WriteString("* 587 RECENT\r\n")
	if status := <-statuses; status.Recent != 587 {
		t.Errorf("Invalid recent count: expected %v but got %v", 587, status.Recent)
	}

	s.WriteString("* 65535 EXPUNGE\r\n")
	if seqNum := <-expunges; seqNum != 65535 {
		t.Errorf("Invalid expunged sequence number: expected %v but got %v", 65535, seqNum)
	}

	s.WriteString("* 431 FETCH (FLAGS (\\Seen))\r\n")
	if msg := <-messages; msg.SeqNum != 431 {
		t.Errorf("Invalid expunged sequence number: expected %v but got %v", 431, msg.SeqNum)
	}

	s.WriteString("* OK Reticulating splines...\r\n")
	if status := <-infos; status.Info != "Reticulating splines..." {
		t.Errorf("Invalid info: got %v", status.Info)
	}

	s.WriteString("* NO Kansai band competition is in 30 seconds !\r\n")
	if status := <-warns; status.Info != "Kansai band competition is in 30 seconds !" {
		t.Errorf("Invalid warning: got %v", status.Info)
	}

	s.WriteString("* BAD Battery level too low, shutting down.\r\n")
	if status := <-errors; status.Info != "Battery level too low, shutting down." {
		t.Errorf("Invalid error: got %v", status.Info)
	}
}
