package server_test

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestCapability(t *testing.T) {
	s, c := testServer(t)
	defer c.Close()
	defer s.Close()

	scanner := bufio.NewScanner(c)
	scanner.Scan() // Greeting

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
	s, c := testServer(t)
	defer c.Close()
	defer s.Close()

	scanner := bufio.NewScanner(c)
	scanner.Scan() // Greeting

	io.WriteString(c, "a001 NOOP\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}
