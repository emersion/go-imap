package server_test

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestLogin_Ok(t *testing.T) {
	s, c := testServer(t)
	defer c.Close()
	defer s.Close()

	scanner := bufio.NewScanner(c)
	scanner.Scan() // Greeting

	io.WriteString(c, "a001 LOGIN username password\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestLogin_No(t *testing.T) {
	s, c := testServer(t)
	defer c.Close()
	defer s.Close()

	scanner := bufio.NewScanner(c)
	scanner.Scan() // Greeting

	io.WriteString(c, "a001 LOGIN username wrongpassword\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}
