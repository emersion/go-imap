package client_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type ClientTester func(c *client.Client) error
type ServerTester func(c net.Conn)

func testClient(t *testing.T, ct ClientTester, st ServerTester) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	done := make(chan error)
	go (func() {
		c, err := client.Dial(l.Addr().String())
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

		c.State = imap.LogoutState
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

type CmdScanner struct {
	scanner *bufio.Scanner
}

func (s *CmdScanner) ScanLine() string {
	s.scanner.Scan()
	return s.scanner.Text()
}

func (s *CmdScanner) Scan() (tag string, cmd string) {
	parts := strings.SplitN(s.ScanLine(), " ", 2)
	return parts[0], parts[1]
}

func NewCmdScanner(r io.Reader) *CmdScanner {
	return &CmdScanner{
		scanner: bufio.NewScanner(r),
	}
}

func removeCmdTag(cmd string) string {
	parts := strings.SplitN(cmd, " ", 2)
	return parts[1]
}

func TestClient(t *testing.T) {
	ct := func(c *client.Client) error {
		if ok, err := c.Support("IMAP4rev1"); err != nil {
			return err
		} else if !ok {
			return errors.New("Server hasn't IMAP4rev1 capability")
		}
		return nil
	}

	st := func(c net.Conn) {}

	testClient(t, ct, st)
}

func TestClient_SetDebug(t *testing.T) {
	ct := func(c *client.Client) error {
		b := &bytes.Buffer{}
		c.SetDebug(b)

		if _, err := c.Capability(); err != nil {
			return err
		}

		if b.Len() == 0 {
			return errors.New("Empty debug buffer")
		}

		return nil
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "CAPABILITY" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "* CAPABILITY IMAP4rev1\r\n")
		io.WriteString(c, tag+" OK CAPABILITY completed.\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_unilateral(t *testing.T) {
	steps := make(chan struct{})

	ct := func(c *client.Client) error {
		c.State = imap.SelectedState
		c.Mailbox = imap.NewMailboxStatus("INBOX", nil)

		statuses := make(chan *imap.MailboxStatus)
		c.MailboxUpdates = statuses
		steps <- struct{}{}

		if status := <-statuses; status.Messages != 42 {
			return fmt.Errorf("Invalid messages count: expected %v but got %v", 42, status.Messages)
		}

		steps <- struct{}{}
		if status := <-statuses; status.Recent != 587 {
			return fmt.Errorf("Invalid recent count: expected %v but got %v", 587, status.Recent)
		}

		expunges := make(chan uint32)
		c.Expunges = expunges
		steps <- struct{}{}
		if seqNum := <-expunges; seqNum != 65535 {
			return fmt.Errorf("Invalid expunged sequence number: expected %v but got %v", 65535, seqNum)
		}

		messages := make(chan *imap.Message)
		c.MessageUpdates = messages
		steps <- struct{}{}
		if msg := <-messages; msg.SeqNum != 431 {
			return fmt.Errorf("Invalid expunged sequence number: expected %v but got %v", 431, msg.SeqNum)
		}

		infos := make(chan *imap.StatusResp)
		c.Infos = infos
		steps <- struct{}{}
		if status := <-infos; status.Info != "Reticulating splines..." {
			return fmt.Errorf("Invalid info: got %v", status.Info)
		}

		warns := make(chan *imap.StatusResp)
		c.Warnings = warns
		steps <- struct{}{}
		if status := <-warns; status.Info != "Kansai band competition is in 30 seconds !" {
			return fmt.Errorf("Invalid warning: got %v", status.Info)
		}

		errors := make(chan *imap.StatusResp)
		c.Errors = errors
		steps <- struct{}{}
		if status := <-errors; status.Info != "Battery level too low, shutting down." {
			return fmt.Errorf("Invalid error: got %v", status.Info)
		}

		return nil
	}

	st := func(c net.Conn) {
		<-steps
		io.WriteString(c, "* 42 EXISTS\r\n")
		<-steps
		io.WriteString(c, "* 587 RECENT\r\n")
		<-steps
		io.WriteString(c, "* 65535 EXPUNGE\r\n")
		<-steps
		io.WriteString(c, "* 431 FETCH (FLAGS (\\Seen))\r\n")
		<-steps
		io.WriteString(c, "* OK Reticulating splines...\r\n")
		<-steps
		io.WriteString(c, "* NO Kansai band competition is in 30 seconds !\r\n")
		<-steps
		io.WriteString(c, "* BAD Battery level too low, shutting down.\r\n")
	}

	testClient(t, ct, st)
}
