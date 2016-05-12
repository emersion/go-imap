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

func TestAuthenticate_Plain_Ok(t *testing.T) {
	s, c := testServer(t)
	defer c.Close()
	defer s.Close()

	scanner := bufio.NewScanner(c)
	scanner.Scan() // Greeting

	io.WriteString(c, "a001 AUTHENTICATE PLAIN\r\n")

	scanner.Scan()
	if scanner.Text() != "+" {
		t.Fatal("Bad continuation request:", scanner.Text())
	}

	// :usename:password
	io.WriteString(c, "AHVzZXJuYW1lAHBhc3N3b3Jk\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestAuthenticate_Plain_No(t *testing.T) {
	s, c := testServer(t)
	defer c.Close()
	defer s.Close()

	scanner := bufio.NewScanner(c)
	scanner.Scan() // Greeting

	io.WriteString(c, "a001 AUTHENTICATE PLAIN\r\n")

	scanner.Scan()
	if scanner.Text() != "+" {
		t.Fatal("Bad continuation request:", scanner.Text())
	}

	// Invalid challenge
	io.WriteString(c, "BHVzZXJuYW1lAHBhc3N3b6Jk\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestAuthenticate_No(t *testing.T) {
	s, c := testServer(t)
	defer c.Close()
	defer s.Close()

	scanner := bufio.NewScanner(c)
	scanner.Scan() // Greeting

	io.WriteString(c, "a001 AUTHENTICATE XIDONTEXIST\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}
