package server_test

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/server"
)

func testServerAuthenticated(t *testing.T) (s *server.Server, c net.Conn, scanner *bufio.Scanner) {
	s, c, scanner = testServerGreeted(t)

	io.WriteString(c, "a000 LOGIN username password\r\n")
	scanner.Scan() // OK response
	return
}

func TestSelect_Ok(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 SELECT INBOX\r\n")

	got := map[string]bool{
		"OK": false,
		"FLAGS": false,
		"EXISTS": false,
		"RECENT": false,
		"UNSEEN": false,
		"PERMANENTFLAGS": false,
		"UIDNEXT": false,
		"UIDVALIDITY": false,
	}

	for scanner.Scan() {
		res := scanner.Text()

		if res == "* FLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft)" {
			got["FLAGS"] = true
		} else if res == "* 1 EXISTS" {
			got["EXISTS"] = true
		} else if res == "* 0 RECENT" {
			got["RECENT"] = true
		} else if strings.HasPrefix(res, "* OK [UNSEEN 0]") {
			got["UNSEEN"] = true
		} else if strings.HasPrefix(res, "* OK [PERMANENTFLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft \\*)]") {
			got["PERMANENTFLAGS"] = true
		} else if strings.HasPrefix(res, "* OK [UIDNEXT 7]") {
			got["UIDNEXT"] = true
		} else if strings.HasPrefix(res, "* OK [UIDVALIDITY 1]") {
			got["UIDVALIDITY"] = true
		} else if strings.HasPrefix(res, "a001 OK [READ-WRITE] ") {
			got["OK"] = true
			break
		} else {
			t.Fatal("Unexpected response:", res)
		}
	}

	for name, val := range got {
		if !val {
			t.Error("Did not got response:", name)
		}
	}
}

func TestSelect_ReadOnly(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 EXAMINE INBOX\r\n")

	gotOk := true
	for scanner.Scan() {
		res := scanner.Text()

		if strings.HasPrefix(res, "a001 OK [READ-ONLY]") {
			gotOk = true
			break
		}
	}

	if !gotOk {
		t.Error("Did not get a correct OK response")
	}
}

func TestSelect_No(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 SELECT idontexist\r\n")

	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCreate(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CREATE test\r\n")
	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestDelete(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CREATE test\r\n")
	scanner.Scan()

	io.WriteString(c, "a001 DELETE test\r\n")
	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestRename(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 CREATE test\r\n")
	scanner.Scan()

	io.WriteString(c, "a001 RENAME test test2\r\n")
	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestSubscribe(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 SUBSCRIBE INBOX\r\n")
	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestUnsubscribe(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 SUBSCRIBE INBOX\r\n")
	scanner.Scan()

	io.WriteString(c, "a001 UNSUBSCRIBE INBOX\r\n")
	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestList(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 LIST \"\" *\r\n")

	scanner.Scan()
	if scanner.Text() != "* LIST (\\Noinferiors) / INBOX" {
		t.Fatal("Invalid LIST response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStatus(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer c.Close()
	defer s.Close()

	io.WriteString(c, "a001 STATUS INBOX (MESSAGES RECENT UIDNEXT UIDVALIDITY UNSEEN)\r\n")

	scanner.Scan()
	if scanner.Text() != "* STATUS INBOX (MESSAGES 1 RECENT 0 UIDNEXT 7 UIDVALIDITY 1 UNSEEN 0)" {
		t.Fatal("Invalid STATUS response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}
