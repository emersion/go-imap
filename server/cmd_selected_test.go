package server_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/server"
)

func testServerSelected(t *testing.T) (s *server.Server, c net.Conn, scanner *bufio.Scanner) {
	s, c, scanner = testServerAuthenticated(t)

	io.WriteString(c, "a000 SELECT INBOX\r\n")

	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "a000 ") {
			break
		}
	}
	return
}

func TestCheck(t *testing.T) {
	s, c, scanner := testServerSelected(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CHECK\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCheck_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CHECK\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestClose(t *testing.T) {
	s, c, scanner := testServerSelected(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CLOSE\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStore(t *testing.T) {
	s, c, scanner := testServerSelected(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 STORE 1 +FLAGS (\\Flagged)\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "* 1 FETCH (FLAGS (\\Seen \\Flagged))") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STORE 1 FLAGS (\\Anwsered)\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "* 1 FETCH (FLAGS (\\Anwsered))") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STORE 1 -FLAGS (\\Anwsered)\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "* 1 FETCH (FLAGS ())") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STORE 1 +FLAGS.SILENT (\\Flagged \\Seen)\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestExpunge(t *testing.T) {
	s, c, scanner := testServerSelected(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 EXPUNGE\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STORE 1 +FLAGS.SILENT (\\Deleted)\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 EXPUNGE\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "* 1 EXPUNGE") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}
