package server_test

import (
	"bufio"
	"net"
	"testing"

	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"
)

// Extnesions that are always advertised by go-imap server.
const builtinExtensions = "LITERAL+ SASL-IR CHILDREN UNSELECT MOVE IDLE APPENDLIMIT"

func testServer(t *testing.T) (s *server.Server, conn net.Conn) {
	bkd := memory.New()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Cannot listen:", err)
	}

	s = server.New(bkd)
	s.AllowInsecureAuth = true

	go s.Serve(l)

	conn, err = net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal("Cannot connect to server:", err)
	}

	return
}

func TestServer_greeting(t *testing.T) {
	s, conn := testServer(t)
	defer s.Close()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	scanner.Scan() // Wait for greeting
	greeting := scanner.Text()

	if greeting != "* OK [CAPABILITY IMAP4rev1 "+builtinExtensions+" AUTH=PLAIN] IMAP4rev1 Service Ready" {
		t.Fatal("Bad greeting:", greeting)
	}
}
