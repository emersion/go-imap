package server_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/server"
	"github.com/emersion/go-sasl"
)

func testServerGreeted(t *testing.T) (s *server.Server, c net.Conn, scanner *bufio.Scanner) {
	s, c = testServer(t)
	scanner = bufio.NewScanner(c)

	scanner.Scan() // Greeting
	return
}

func TestCapability(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CAPABILITY\r\n")

	scanner.Scan()
	if scanner.Text() != "* CAPABILITY IMAP4rev1 "+builtinExtensions+" AUTH=PLAIN" {
		t.Fatal("Bad capability:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestNoop(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 NOOP\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestLogout(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

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

type xnoop struct{}

func (ext *xnoop) Capabilities(server.Conn) []string {
	return []string{"XNOOP"}
}

func (ext *xnoop) Command(string) server.HandlerFactory {
	return nil
}

func TestServer_Enable(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	s.Enable(&xnoop{})

	io.WriteString(c, "a001 CAPABILITY\r\n")

	scanner.Scan()
	if scanner.Text() != "* CAPABILITY IMAP4rev1 "+builtinExtensions+" AUTH=PLAIN XNOOP" {
		t.Fatal("Bad capability:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

type xnoopAuth struct{}

func (ext *xnoopAuth) Next(response []byte) (challenge []byte, done bool, err error) {
	done = true
	return
}

func TestServer_EnableAuth(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	s.EnableAuth("XNOOP", func(server.Conn) sasl.Server {
		return &xnoopAuth{}
	})

	io.WriteString(c, "a001 CAPABILITY\r\n")

	scanner.Scan()
	if scanner.Text() != "* CAPABILITY IMAP4rev1 "+builtinExtensions+" AUTH=PLAIN AUTH=XNOOP" &&
		scanner.Text() != "* CAPABILITY IMAP4rev1 "+builtinExtensions+" AUTH=XNOOP AUTH=PLAIN" {
		t.Fatal("Bad capability:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}
