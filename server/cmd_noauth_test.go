package server_test

import (
	"bufio"
	"crypto/tls"
	"io"
	"strings"
	"testing"

	"github.com/emersion/go-imap/internal"
)

func TestStartTLS(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

	cert, err := tls.X509KeyPair(internal.LocalhostCert, internal.LocalhostKey)
	if err != nil {
		t.Fatal(err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	s.AllowInsecureAuth = false
	s.TLSConfig = tlsConfig

	io.WriteString(c, "a001 CAPABILITY\r\n")
	scanner.Scan()
	if scanner.Text() != "* CAPABILITY IMAP4rev1 STARTTLS LOGINDISABLED" {
		t.Fatal("Bad CAPABILITY response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STARTTLS\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
	sc := tls.Client(c, tlsConfig)
	if err = sc.Handshake(); err != nil {
		t.Fatal(err)
	}
	scanner = bufio.NewScanner(sc)

	scanner.Scan()
	if scanner.Text() != "* CAPABILITY IMAP4rev1 AUTH=PLAIN" {
		t.Fatal("Bad CAPABILITY response:", scanner.Text())
	}
}

func TestLogin_Ok(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 LOGIN username password\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestLogin_No(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 LOGIN username wrongpassword\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestAuthenticate_Plain_Ok(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

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
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

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
	s, c, scanner := testServerGreeted(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 AUTHENTICATE XIDONTEXIST\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}
