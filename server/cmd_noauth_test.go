package server_test

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/internal"
	"github.com/emersion/go-imap/server"
)

func testServerTLS(t *testing.T) (s *server.Server, c net.Conn, scanner *bufio.Scanner) {
	s, c, scanner = testServerGreeted(t)

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
	if scanner.Text() != "* CAPABILITY IMAP4rev1 "+builtinExtensions+" STARTTLS LOGINDISABLED" {
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

	c = sc
	scanner = bufio.NewScanner(c)
	return
}

func TestStartTLS(t *testing.T) {
	s, c, scanner := testServerTLS(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CAPABILITY\r\n")

	scanner.Scan()
	if scanner.Text() != "* CAPABILITY IMAP4rev1 "+builtinExtensions+" AUTH=PLAIN" {
		t.Fatal("Bad CAPABILITY response:", scanner.Text())
	}
}

func TestStartTLS_AlreadyEnabled(t *testing.T) {
	s, c, scanner := testServerTLS(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STARTTLS\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestStartTLS_NotSupported(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STARTTLS\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestLogin_Ok(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 LOGIN username password\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestLogin_AlreadyAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 LOGIN username password\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}

	io.WriteString(c, "a001 LOGIN username password\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestLogin_No(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 LOGIN username wrongpassword\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestAuthenticate_Plain_Ok(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

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

func TestAuthenticate_Plain_Cancel(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 AUTHENTICATE PLAIN\r\n")

	scanner.Scan()
	if scanner.Text() != "+" {
		t.Fatal("Bad continuation request:", scanner.Text())
	}

	io.WriteString(c, "*\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 BAD negotiation cancelled") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestAuthenticate_Plain_InitialResponse(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 AUTHENTICATE PLAIN AHVzZXJuYW1lAHBhc3N3b3Jk\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestAuthenticate_Plain_No(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

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
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 AUTHENTICATE XIDONTEXIST\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}
