package client_test

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/imap/client"
)

type ClientTester func(c *client.Client) error
type ServerTester func(c net.Conn)

func testClient(t *testing.T, ct ClientTester, st ServerTester) {
	addr := ":3000"

	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	done := make(chan error)
	go (func () {
		c, err := client.Dial(addr)
		if err != nil {
			done <- err
			return
		}

		err = ct(c)
		if err != nil {
			done <- err
			return
		}

		done <- nil

		c.Logout()
	})()

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	greeting := "* OK [CAPABILITY IMAP4rev1] Server ready.\r\n"
	if _, err = io.WriteString(conn, greeting); err != nil {
		t.Fatal(err)
	}

	st(conn)

	err = <-done
	if err != nil {
		t.Fatal(err)
	}

	io.WriteString(conn, "* BYE Shutting down.\r\n")
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
		if !c.Caps["IMAP4rev1"] {
			return errors.New("Server hasn't IMAP4rev1 capability")
		}
		return nil
	}

	st := func(c net.Conn) {}

	testClient(t, ct, st)
}
