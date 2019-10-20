package server_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/server"
)

func testServerSelected(t *testing.T, readOnly bool) (s *server.Server, c net.Conn, scanner *bufio.Scanner) {
	s, c, scanner = testServerAuthenticated(t)

	if readOnly {
		io.WriteString(c, "a000 EXAMINE INBOX\r\n")
	} else {
		io.WriteString(c, "a000 SELECT INBOX\r\n")
	}

	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "a000 ") {
			break
		}
	}
	return
}

func TestNoop_Selected(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 NOOP\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Bad status response:", scanner.Text())
	}
}

func TestCheck(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CHECK\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCheck_ReadOnly(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CHECK\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCheck_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CHECK\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestClose(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CLOSE\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestClose_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CLOSE\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestExpunge(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

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

func TestExpunge_ReadOnly(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 EXPUNGE\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestExpunge_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 EXPUNGE\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestSearch(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

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

func TestSearch_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 SEARCH UNDELETED\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestSearch_Uid(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 UID SEARCH UNDELETED\r\n")
	scanner.Scan()
	if scanner.Text() != "* SEARCH 6" {
		t.Fatal("Invalid SEARCH response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestFetch(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

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

func TestFetch_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 FETCH 1 (UID FLAGS)\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestFetch_Uid(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 UID FETCH 6 (UID)\r\n")
	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (UID 6)" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestFetch_Uid_UidNotRequested(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 UID FETCH 6 (FLAGS)\r\n")
	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Seen) UID 6)" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStore(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STORE 1 +FLAGS (\\Flagged)\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Seen \\Flagged))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STORE 1 FLAGS (\\Answered)\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Answered))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 STORE 1 -FLAGS (\\Answered)\r\n")

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

func TestStore_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STORE 1 +FLAGS (\\Flagged)\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStore_ReadOnly(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STORE 1 +FLAGS (\\Flagged)\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStore_InvalidOperation(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STORE 1 IDONTEXIST (\\Flagged)\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStore_InvalidFlags(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STORE 1 +FLAGS ((nested)(lists))\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStore_SingleFlagNonList(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 STORE 1 FLAGS somestring\r\n")

	gotOK := false
	gotFetch := false
	for scanner.Scan() {
		res := scanner.Text()

		if res == "* 1 FETCH (FLAGS (somestring))" {
			gotFetch = true
		} else if strings.HasPrefix(res, "a001 OK ") {
			gotOK = true
			break
		} else {
			t.Fatal("Unexpected response:", res)
		}
	}

	if !gotFetch {
		t.Fatal("Missing FETCH response.")
	}

	if !gotOK {
		t.Fatal("Missing status response.")
	}
}

func TestStore_NonList(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 STORE 1 FLAGS somestring someanotherstring\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (somestring someanotherstring))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStore_RecentFlag(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer c.Close()
	defer s.Close()

	// Add Recent flag
	io.WriteString(c, "a001 STORE 1 FLAGS \\Recent\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Recent))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	// Set flags to: something
	// Should still get Recent flag back
	io.WriteString(c, "a001 STORE 1 FLAGS something\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Recent something))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	// Try adding Recent flag again
	io.WriteString(c, "a001 STORE 1 FLAGS \\Recent anotherflag\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Recent anotherflag))" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStore_Uid(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 UID STORE 6 +FLAGS (\\Flagged)\r\n")

	scanner.Scan()
	if scanner.Text() != "* 1 FETCH (FLAGS (\\Seen \\Flagged) UID 6)" {
		t.Fatal("Invalid FETCH response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCopy(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

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
	if !strings.HasPrefix(scanner.Text(), "* STATUS \"CopyDest\" (MESSAGES 1)") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCopy_NotSelected(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CREATE CopyDest\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 COPY 1 CopyDest\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCopy_Uid(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CREATE CopyDest\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 UID COPY 6 CopyDest\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestUid_InvalidCommand(t *testing.T) {
	s, c, scanner := testServerSelected(t, false)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 UID IDONTEXIST\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 UID CLOSE\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}
