package client

import (
	"bufio"
	"bytes"
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
	net.Conn
	net.Listener
}

func (c *serverConn) Close() error {
	if err := c.Conn.Close(); err != nil {
		return err
	}
	return c.Listener.Close()
}

func (c *serverConn) WriteString(s string) (n int, err error) {
	return io.WriteString(c.Conn, s)
}

func newTestClient(t *testing.T) (c *Client, s *serverConn) {
	return newTestClientWithGreeting(t, "* OK [CAPABILITY IMAP4rev1 STARTTLS AUTH=PLAIN UNSELECT] Server ready.\r\n")
}

func newTestClientWithGreeting(t *testing.T, greeting string) (c *Client, s *serverConn) {
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

		if _, err := io.WriteString(conn, greeting); err != nil {
			panic(err)
		}

		s = &serverConn{newCmdScanner(conn), conn, l}
		close(done)
	}()

	c, err = Dial(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	<-done
	return
}

func setClientState(c *Client, state imap.ConnState, mailbox *imap.MailboxStatus) {
	c.locker.Lock()
	c.state = state
	c.mailbox = mailbox
	c.locker.Unlock()
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
	done := make(chan error)

	go func() {
		c.SetDebug(&b)
		done <- nil
	}()
	if tag, cmd := s.ScanCmd(); cmd != "NOOP" {
		t.Fatal("Bad command:", cmd)
	} else {
		s.WriteString(tag + " OK NOOP completed.\r\n")
	}
	// wait for SetDebug to finish.
	<-done

	go func() {
		_, err := c.Capability()
		done <- err
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "CAPABILITY" {
		t.Fatal("Bad command:", cmd)
	}

	s.WriteString("* CAPABILITY IMAP4rev1\r\n")
	s.WriteString(tag + " OK CAPABILITY completed.\r\n")

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

	setClientState(c, imap.SelectedState, imap.NewMailboxStatus("INBOX", nil))

	updates := make(chan Update, 1)
	c.Updates = updates

	s.WriteString("* 42 EXISTS\r\n")
	if update, ok := (<-updates).(*MailboxUpdate); !ok || update.Mailbox.Messages != 42 {
		t.Errorf("Invalid messages count: expected %v but got %v", 42, update.Mailbox.Messages)
	}

	s.WriteString("* 587 RECENT\r\n")
	if update, ok := (<-updates).(*MailboxUpdate); !ok || update.Mailbox.Recent != 587 {
		t.Errorf("Invalid recent count: expected %v but got %v", 587, update.Mailbox.Recent)
	}

	s.WriteString("* 65535 EXPUNGE\r\n")
	if update, ok := (<-updates).(*ExpungeUpdate); !ok || update.SeqNum != 65535 {
		t.Errorf("Invalid expunged sequence number: expected %v but got %v", 65535, update.SeqNum)
	}

	s.WriteString("* 431 FETCH (FLAGS (\\Seen))\r\n")
	if update, ok := (<-updates).(*MessageUpdate); !ok || update.Message.SeqNum != 431 {
		t.Errorf("Invalid expunged sequence number: expected %v but got %v", 431, update.Message.SeqNum)
	}

	s.WriteString("* OK Reticulating splines...\r\n")
	if update, ok := (<-updates).(*StatusUpdate); !ok || update.Status.Info != "Reticulating splines..." {
		t.Errorf("Invalid info: got %v", update.Status.Info)
	}

	s.WriteString("* NO Kansai band competition is in 30 seconds !\r\n")
	if update, ok := (<-updates).(*StatusUpdate); !ok || update.Status.Info != "Kansai band competition is in 30 seconds !" {
		t.Errorf("Invalid warning: got %v", update.Status.Info)
	}

	s.WriteString("* BAD Battery level too low, shutting down.\r\n")
	if update, ok := (<-updates).(*StatusUpdate); !ok || update.Status.Info != "Battery level too low, shutting down." {
		t.Errorf("Invalid error: got %v", update.Status.Info)
	}
}
