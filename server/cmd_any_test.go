package server_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/server"
)

func testServerGreeted(t *testing.T) (s *server.Server, c net.Conn, scanner *bufio.Scanner) {
	s, c = testServer(t)
	scanner = bufio.NewScanner(c)

	scanner.Scan() // Greeting
	return
}

func TestCapability(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CAPABILITY\r\n")

	scanner.Scan()
	if scanner.Text() != "* CAPABILITY IMAP4rev1 AUTH=PLAIN" {
		t.Fatal("Bad capability:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestNoop(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 NOOP\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestLogout(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 LOGOUT\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "* BYE ") {
		t.Fatal("Bad BYE response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}
