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

		c.Logout()
		done <- nil
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

	io.WriteString(conn, "* BYE Shutting down.\r\n")
	conn.Close()

	err = <-done
	if err != nil {
		t.Fatal(err)
	}
}

type CmdScanner struct {
	scanner *bufio.Scanner
}

func (s *CmdScanner) Scan() (tag string, cmd string) {
	s.scanner.Scan()

	parts := strings.SplitN(s.scanner.Text(), " ", 2)
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
	ct := func(c *client.Client) (err error) {
		if !c.Caps["IMAP4rev1"] {
			err = errors.New("Server hasn't IMAP4rev1 capability")
		}
		return
	}

	st := func(c net.Conn) {}

	testClient(t, ct, st)
}
