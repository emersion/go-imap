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

func TestClose_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CLOSE\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
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
	if scanner.Text() != "* 1 EXPUNGE" {
		t.Fatal("Invalid EXPUNGE response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestSearch(t *testing.T) {
	s, c, scanner := testServerSelected(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 SEARCH UNDELETED\r\n")
	scanner.Scan()
	if scanner.Text() != "* SEARCH 1" {
		t.Fatal("Invalid SEARCH response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 SEARCH DELETED\r\n")
	scanner.Scan()
	if scanner.Text() != "* SEARCH" {
		t.Fatal("Invalid SEARCH response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestFetch(t *testing.T) {
	s, c, scanner := testServerSelected(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 FETCH 1 (UID FLAGS)\r\n")
	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (UID 6 FLAGS (\\Seen))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 FETCH 1 (BODY.PEEK[TEXT])\r\n")
	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (BODY[TEXT] {11}" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "Hi there :))") {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}
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
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Seen \\Flagged))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STORE 1 FLAGS (\\Anwsered)\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Anwsered))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STORE 1 -FLAGS (\\Anwsered)\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS ())" {
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

func TestCopy(t *testing.T) {
	s, c, scanner := testServerSelected(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CREATE CopyDest\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 COPY 1 CopyDest\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STATUS CopyDest (MESSAGES)\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "* STATUS CopyDest (MESSAGES 1)") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}
