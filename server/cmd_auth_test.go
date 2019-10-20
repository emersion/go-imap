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
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 SELECT INBOX\r\n")

	got := map[string]bool{
		"OK":             false,
		"FLAGS":          false,
		"EXISTS":         false,
		"RECENT":         false,
		"PERMANENTFLAGS": false,
		"UIDNEXT":        false,
		"UIDVALIDITY":    false,
	}

	for scanner.Scan() {
		res := scanner.Text()

		if res == "* FLAGS (\\Seen)" {
			got["FLAGS"] = true
		} else if res == "* 1 EXISTS" {
			got["EXISTS"] = true
		} else if res == "* 0 RECENT" {
			got["RECENT"] = true
		} else if strings.HasPrefix(res, "* OK [PERMANENTFLAGS (\\*)]") {
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
	defer s.Close()
	defer c.Close()

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

func TestSelect_InvalidMailbox(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 SELECT idontexist\r\n")

	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestSelect_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 SELECT INBOX\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCreate(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CREATE test\r\n")
	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestCreate_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CREATE test\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestDelete(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CREATE test\r\n")
	scanner.Scan()

	io.WriteString(c, "a001 DELETE test\r\n")
	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestDelete_InvalidMailbox(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 DELETE test\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestDelete_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 DELETE INBOX\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestRename(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CREATE test\r\n")
	scanner.Scan()

	io.WriteString(c, "a001 RENAME test test2\r\n")
	scanner.Scan()

	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestRename_InvalidMailbox(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 RENAME test test2\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestRename_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 RENAME test test2\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestSubscribe(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 SUBSCRIBE INBOX\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 SUBSCRIBE idontexist\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestSubscribe_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 SUBSCRIBE INBOX\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestUnsubscribe(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 SUBSCRIBE INBOX\r\n")
	scanner.Scan()

	io.WriteString(c, "a001 UNSUBSCRIBE INBOX\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 UNSUBSCRIBE idontexist\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestUnsubscribe_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 UNSUBSCRIBE INBOX\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestList(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 LIST \"\" *\r\n")

	scanner.Scan()
	if scanner.Text() != "* LIST () \"/\" INBOX" {
		t.Fatal("Invalid LIST response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestList_Nested(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 CREATE first\r\n")
	scanner.Scan()
	io.WriteString(c, "a001 CREATE first/second\r\n")
	scanner.Scan()
	io.WriteString(c, "a001 CREATE first/second/third\r\n")
	scanner.Scan()
	io.WriteString(c, "a001 CREATE first/second/third2\r\n")
	scanner.Scan()

	check := func(mailboxes []string) {
		checked := map[string]bool{}

		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "a001 OK ") {
				break
			} else if strings.HasPrefix(scanner.Text(), "* LIST ") {
				found := false
				for _, name := range mailboxes {
					if strings.HasSuffix(scanner.Text(), " \""+name+"\"") || strings.HasSuffix(scanner.Text(), " "+name) {
						checked[name] = true
						found = true
						break
					}
				}

				if !found {
					t.Fatal("Unexpected mailbox:", scanner.Text())
				}
			} else {
				t.Fatal("Invalid LIST response:", scanner.Text())
			}
		}

		for _, name := range mailboxes {
			if !checked[name] {
				t.Fatal("Missing mailbox:", name)
			}
		}
	}

	io.WriteString(c, "a001 LIST \"\" *\r\n")
	check([]string{"INBOX", "first", "first/second", "first/second/third", "first/second/third2"})

	io.WriteString(c, "a001 LIST \"\" %\r\n")
	check([]string{"INBOX", "first"})

	io.WriteString(c, "a001 LIST first *\r\n")
	check([]string{"first/second", "first/second/third", "first/second/third2"})

	io.WriteString(c, "a001 LIST first %\r\n")
	check([]string{"first/second"})

	io.WriteString(c, "a001 LIST first/second *\r\n")
	check([]string{"first/second/third", "first/second/third2"})

	io.WriteString(c, "a001 LIST first/second %\r\n")
	check([]string{"first/second/third", "first/second/third2"})

	io.WriteString(c, "a001 LIST first second\r\n")
	check([]string{"first/second"})

	io.WriteString(c, "a001 LIST first/second third\r\n")
	check([]string{"first/second/third"})
}

func TestList_Subscribed(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 LSUB \"\" *\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}

	io.WriteString(c, "a001 SUBSCRIBE INBOX\r\n")
	scanner.Scan()

	io.WriteString(c, "a001 LSUB \"\" *\r\n")

	scanner.Scan()
	if scanner.Text() != "* LSUB () \"/\" INBOX" {
		t.Fatal("Invalid LSUB response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestTLS_AlreadyAuthenticated(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STARTTLS\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestList_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 LIST \"\" *\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestList_Delimiter(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 LIST \"\" \"\"\r\n")

	scanner.Scan()
	if scanner.Text() != "* LIST (\\Noselect) \"/\" \"/\"" {
		t.Fatal("Invalid LIST response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStatus(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STATUS INBOX (MESSAGES RECENT UIDNEXT UIDVALIDITY UNSEEN)\r\n")

	scanner.Scan()
	line := scanner.Text()
	if !strings.HasPrefix(line, "* STATUS INBOX (") {
		t.Fatal("Invalid STATUS response:", line)
	}
	parts := []string{"MESSAGES 1", "RECENT 0", "UIDNEXT 7", "UIDVALIDITY 1", "UNSEEN 0"}
	for _, p := range parts {
		if !strings.Contains(line, p) {
			t.Fatal("Invalid STATUS response:", line)
		}
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStatus_InvalidMailbox(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STATUS idontexist (MESSAGES)\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestStatus_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 STATUS INBOX (MESSAGES)\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestAppend(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 APPEND INBOX {80}\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "+ ") {
		t.Fatal("Invalid continuation request:", scanner.Text())
	}

	io.WriteString(c, "From: Edward Snowden <root@nsa.gov>\r\n")
	io.WriteString(c, "To: Julian Assange <root@gchq.gov.uk>\r\n")
	io.WriteString(c, "\r\n")
	io.WriteString(c, "<3\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestAppend_WithFlags(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 APPEND INBOX (\\Draft) {11}\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "+ ") {
		t.Fatal("Invalid continuation request:", scanner.Text())
	}

	io.WriteString(c, "Hello World\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestAppend_WithFlagsAndDate(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 APPEND INBOX (\\Draft) \"5-Nov-1984 13:37:00 -0700\" {11}\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "+ ") {
		t.Fatal("Invalid continuation request:", scanner.Text())
	}

	io.WriteString(c, "Hello World\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestAppend_Selected(t *testing.T) {
	s, c, scanner := testServerSelected(t, true)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 APPEND INBOX {11}\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "+ ") {
		t.Fatal("Invalid continuation request:", scanner.Text())
	}

	io.WriteString(c, "Hello World\r\n")

	scanner.Scan()
	if scanner.Text() != "* 2 EXISTS" {
		t.Fatal("Invalid untagged response:", scanner.Text())
	}

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 OK ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestAppend_InvalidMailbox(t *testing.T) {
	s, c, scanner := testServerAuthenticated(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 APPEND idontexist {11}\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "+ ") {
		t.Fatal("Invalid continuation request:", scanner.Text())
	}

	io.WriteString(c, "Hello World\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}

func TestAppend_NotAuthenticated(t *testing.T) {
	s, c, scanner := testServerGreeted(t)
	defer s.Close()
	defer c.Close()

	io.WriteString(c, "a001 APPEND INBOX {11}\r\n")
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "+ ") {
		t.Fatal("Invalid continuation request:", scanner.Text())
	}

	io.WriteString(c, "Hello World\r\n")

	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), "a001 NO ") {
		t.Fatal("Invalid status response:", scanner.Text())
	}
}
