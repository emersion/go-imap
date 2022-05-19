package client

import (
	"io/ioutil"
	"net/textproto"
	"reflect"
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

func TestClient_Check(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	done := make(chan error, 1)
	go func() {
		done <- c.Check()
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "CHECK" {
		t.Fatalf("client sent command %v, want %v", cmd, "CHECK")
	}

	s.WriteString(tag + " OK CHECK completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Check() = %v", err)
	}
}

func TestClient_Close(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, &imap.MailboxStatus{Name: "INBOX"})

	done := make(chan error, 1)
	go func() {
		done <- c.Close()
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "CLOSE" {
		t.Fatalf("client sent command %v, want %v", cmd, "CLOSE")
	}

	s.WriteString(tag + " OK CLOSE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Check() = %v", err)
	}

	if state := c.State(); state != imap.AuthenticatedState {
		t.Errorf("Bad state: %v", state)
	}
	if mailbox := c.Mailbox(); mailbox != nil {
		t.Errorf("Client selected mailbox is not nil: %v", mailbox)
	}
}

func TestClient_Expunge(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	done := make(chan error, 1)
	expunged := make(chan uint32, 4)
	go func() {
		done <- c.Expunge(expunged)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "EXPUNGE" {
		t.Fatalf("client sent command %v, want %v", cmd, "EXPUNGE")
	}

	s.WriteString("* 3 EXPUNGE\r\n")
	s.WriteString("* 3 EXPUNGE\r\n")
	s.WriteString("* 5 EXPUNGE\r\n")
	s.WriteString("* 8 EXPUNGE\r\n")
	s.WriteString(tag + " OK EXPUNGE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Expunge() = %v", err)
	}

	expected := []uint32{3, 3, 5, 8}

	i := 0
	for id := range expunged {
		if id != expected[i] {
			t.Errorf("Bad expunged sequence number: got %v instead of %v", id, expected[i])
		}
		i++
	}
}

func TestClient_Search(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	date, _ := time.Parse(imap.DateLayout, "1-Feb-1994")
	criteria := &imap.SearchCriteria{
		WithFlags: []string{imap.DeletedFlag},
		Header:    textproto.MIMEHeader{"From": {"Smith"}},
		Since:     date,
		Not: []*imap.SearchCriteria{{
			Header: textproto.MIMEHeader{"To": {"Pauline"}},
		}},
	}

	done := make(chan error, 1)
	var results []uint32
	go func() {
		var err error
		results, err = c.Search(criteria)
		done <- err
	}()

	wantCmd := `SEARCH CHARSET UTF-8 SINCE "1-Feb-1994" FROM "Smith" DELETED NOT (TO "Pauline")`
	tag, cmd := s.ScanCmd()
	if cmd != wantCmd {
		t.Fatalf("client sent command %v, want %v", cmd, wantCmd)
	}

	s.WriteString("* SEARCH 2 84 882\r\n")
	s.WriteString(tag + " OK SEARCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Search() = %v", err)
	}

	want := []uint32{2, 84, 882}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("c.Search() = %v, want %v", results, want)
	}
}

func TestClient_Search_Badcharset(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	date, _ := time.Parse(imap.DateLayout, "1-Feb-1994")
	criteria := &imap.SearchCriteria{
		WithFlags: []string{imap.DeletedFlag},
		Header:    textproto.MIMEHeader{"From": {"Smith"}},
		Since:     date,
		Not: []*imap.SearchCriteria{{
			Header: textproto.MIMEHeader{"To": {"Pauline"}},
		}},
	}

	done := make(chan error, 1)
	var results []uint32
	go func() {
		var err error
		results, err = c.Search(criteria)
		done <- err
	}()

	// first search call with default UTF-8 charset (assume server does not support UTF-8)
	wantCmd := `SEARCH CHARSET UTF-8 SINCE "1-Feb-1994" FROM "Smith" DELETED NOT (TO "Pauline")`
	tag, cmd := s.ScanCmd()
	if cmd != wantCmd {
		t.Fatalf("client sent command %v, want %v", cmd, wantCmd)
	}

	s.WriteString(tag + " NO [BADCHARSET (US-ASCII)]\r\n")

	// internal fall-back to US-ASCII which sets utf8SearchUnsupported = true
	wantCmd = `SEARCH CHARSET US-ASCII SINCE "1-Feb-1994" FROM "Smith" DELETED NOT (TO "Pauline")`
	tag, cmd = s.ScanCmd()
	if cmd != wantCmd {
		t.Fatalf("client sent command %v, want %v", cmd, wantCmd)
	}

	s.WriteString("* SEARCH 2 84 882\r\n")
	s.WriteString(tag + " OK SEARCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("err: %v", err)
	}

	want := []uint32{2, 84, 882}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("c.Search() = %v, want %v", results, want)
	}

	if !c.utf8SearchUnsupported {
		t.Fatal("client should have utf8SearchUnsupported set to true")
	}

	// second call to search (with utf8SearchUnsupported=true)
	go func() {
		var err error
		results, err = c.Search(criteria)
		done <- err
	}()

	tag, cmd = s.ScanCmd()
	if cmd != wantCmd {
		t.Fatalf("client sent command %v, want %v", cmd, wantCmd)
	}

	s.WriteString("* SEARCH 2 84 882\r\n")
	s.WriteString(tag + " OK SEARCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Search() = %v", err)
	}

	if !reflect.DeepEqual(results, want) {
		t.Errorf("c.Search() = %v, want %v", results, want)
	}
}

func TestClient_Search_Uid(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	criteria := &imap.SearchCriteria{
		WithoutFlags: []string{imap.DeletedFlag},
	}

	done := make(chan error, 1)
	var results []uint32
	go func() {
		var err error
		results, err = c.UidSearch(criteria)
		done <- err
	}()

	wantCmd := "UID SEARCH CHARSET UTF-8 UNDELETED"
	tag, cmd := s.ScanCmd()
	if cmd != wantCmd {
		t.Fatalf("client sent command %v, want %v", cmd, wantCmd)
	}

	s.WriteString("* SEARCH 1 78 2010\r\n")
	s.WriteString(tag + " OK UID SEARCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Search() = %v", err)
	}

	want := []uint32{1, 78, 2010}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("c.Search() = %v, want %v", results, want)
	}
}

func TestClient_Search_Uid_Badcharset(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	criteria := &imap.SearchCriteria{
		WithoutFlags: []string{imap.DeletedFlag},
	}

	done := make(chan error, 1)
	var results []uint32
	go func() {
		var err error
		results, err = c.UidSearch(criteria)
		done <- err
	}()

	// first search call with default UTF-8 charset (assume server does not support UTF-8)
	wantCmd := "UID SEARCH CHARSET UTF-8 UNDELETED"
	tag, cmd := s.ScanCmd()
	if cmd != wantCmd {
		t.Fatalf("client sent command %v, want %v", cmd, wantCmd)
	}

	s.WriteString(tag + " NO [BADCHARSET (US-ASCII)]\r\n")

	// internal fall-back to US-ASCII which sets utf8SearchUnsupported = true
	wantCmd = "UID SEARCH CHARSET US-ASCII UNDELETED"
	tag, cmd = s.ScanCmd()
	if cmd != wantCmd {
		t.Fatalf("client sent command %v, want %v", cmd, wantCmd)
	}

	s.WriteString("* SEARCH 1 78 2010\r\n")
	s.WriteString(tag + " OK UID SEARCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.UidSearch() = %v", err)
	}

	want := []uint32{1, 78, 2010}
	if !reflect.DeepEqual(results, want) {
		t.Errorf("c.UidSearch() = %v, want %v", results, want)
	}

	if !c.utf8SearchUnsupported {
		t.Fatal("client should have utf8SearchUnsupported set to true")
	}

	// second call to search (with utf8SearchUnsupported=true)
	go func() {
		var err error
		results, err = c.UidSearch(criteria)
		done <- err
	}()

	tag, cmd = s.ScanCmd()
	if cmd != wantCmd {
		t.Fatalf("client sent command %v, want %v", cmd, wantCmd)
	}

	s.WriteString("* SEARCH 1 78 2010\r\n")
	s.WriteString(tag + " OK UID SEARCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.UidSearch() = %v", err)
	}

	if !reflect.DeepEqual(results, want) {
		t.Errorf("c.UidSearch() = %v, want %v", results, want)
	}

}

func TestClient_Fetch(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("2:3")
	fields := []imap.FetchItem{imap.FetchUid, imap.FetchItem("BODY[]")}

	done := make(chan error, 1)
	messages := make(chan *imap.Message, 2)
	go func() {
		done <- c.Fetch(seqset, fields, messages)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "FETCH 2:3 (UID BODY[])" {
		t.Fatalf("client sent command %v, want %v", cmd, "FETCH 2:3 (UID BODY[])")
	}

	s.WriteString("* 2 FETCH (UID 42 BODY[] {16}\r\n")
	s.WriteString("I love potatoes.")
	s.WriteString(")\r\n")

	s.WriteString("* 3 FETCH (UID 28 BODY[] {12}\r\n")
	s.WriteString("Hello World!")
	s.WriteString(")\r\n")

	s.WriteString(tag + " OK FETCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Fetch() = %v", err)
	}

	section, _ := imap.ParseBodySectionName("BODY[]")

	msg := <-messages
	if msg.SeqNum != 2 {
		t.Errorf("First message has bad sequence number: %v", msg.SeqNum)
	}
	if msg.Uid != 42 {
		t.Errorf("First message has bad UID: %v", msg.Uid)
	}
	if body, _ := ioutil.ReadAll(msg.GetBody(section)); string(body) != "I love potatoes." {
		t.Errorf("First message has bad body: %q", body)
	}

	msg = <-messages
	if msg.SeqNum != 3 {
		t.Errorf("First message has bad sequence number: %v", msg.SeqNum)
	}
	if msg.Uid != 28 {
		t.Errorf("Second message has bad UID: %v", msg.Uid)
	}
	if body, _ := ioutil.ReadAll(msg.GetBody(section)); string(body) != "Hello World!" {
		t.Errorf("Second message has bad body: %q", body)
	}
}

func TestClient_Fetch_ClosedState(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	seqset, _ := imap.ParseSeqSet("2:3")
	fields := []imap.FetchItem{imap.FetchUid, imap.FetchItem("BODY[]")}

	done := make(chan error, 1)
	messages := make(chan *imap.Message, 2)
	go func() {
		done <- c.Fetch(seqset, fields, messages)
	}()

	_, more := <-messages

	if more {
		t.Fatalf("Messages channel has more messages, but it must be closed with no messages sent")
	}

	err := <-done

	if err != ErrNoMailboxSelected {
		t.Fatalf("Expected error to be IMAP Client ErrNoMailboxSelected")
	}
}

func TestClient_Fetch_Partial(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("1")
	fields := []imap.FetchItem{imap.FetchItem("BODY.PEEK[]<0.10>")}

	done := make(chan error, 1)
	messages := make(chan *imap.Message, 1)
	go func() {
		done <- c.Fetch(seqset, fields, messages)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "FETCH 1 (BODY.PEEK[]<0.10>)" {
		t.Fatalf("client sent command %v, want %v", cmd, "FETCH 1 (BODY.PEEK[]<0.10>)")
	}

	s.WriteString("* 1 FETCH (BODY[]<0> {10}\r\n")
	s.WriteString("I love pot")
	s.WriteString(")\r\n")

	s.WriteString(tag + " OK FETCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Fetch() = %v", err)
	}

	section, _ := imap.ParseBodySectionName("BODY.PEEK[]<0.10>")

	msg := <-messages
	if body, _ := ioutil.ReadAll(msg.GetBody(section)); string(body) != "I love pot" {
		t.Errorf("Message has bad body: %q", body)
	}
}

func TestClient_Fetch_part(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("1")
	fields := []imap.FetchItem{imap.FetchItem("BODY.PEEK[1]")}

	done := make(chan error, 1)
	messages := make(chan *imap.Message, 1)
	go func() {
		done <- c.Fetch(seqset, fields, messages)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "FETCH 1 (BODY.PEEK[1])" {
		t.Fatalf("client sent command %v, want %v", cmd, "FETCH 1 (BODY.PEEK[1])")
	}

	s.WriteString("* 1 FETCH (BODY[1] {3}\r\n")
	s.WriteString("Hey")
	s.WriteString(")\r\n")

	s.WriteString(tag + " OK FETCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Fetch() = %v", err)
	}

	<-messages
}

func TestClient_Fetch_Uid(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("1:867")
	fields := []imap.FetchItem{imap.FetchFlags}

	done := make(chan error, 1)
	messages := make(chan *imap.Message, 1)
	go func() {
		done <- c.UidFetch(seqset, fields, messages)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "UID FETCH 1:867 (FLAGS)" {
		t.Fatalf("client sent command %v, want %v", cmd, "UID FETCH 1:867 (FLAGS)")
	}

	s.WriteString("* 23 FETCH (UID 42 FLAGS (\\Seen))\r\n")
	s.WriteString(tag + " OK UID FETCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.UidFetch() = %v", err)
	}

	msg := <-messages
	if msg.SeqNum != 23 {
		t.Errorf("First message has bad sequence number: %v", msg.SeqNum)
	}
	if msg.Uid != 42 {
		t.Errorf("Message has bad UID: %v", msg.Uid)
	}
	if len(msg.Flags) != 1 || msg.Flags[0] != "\\Seen" {
		t.Errorf("Message has bad flags: %v", msg.Flags)
	}
}

func TestClient_Fetch_Unilateral(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("1:4")
	fields := []imap.FetchItem{imap.FetchFlags}

	done := make(chan error, 1)
	messages := make(chan *imap.Message, 3)
	go func() {
		done <- c.Fetch(seqset, fields, messages)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "FETCH 1:4 (FLAGS)" {
		t.Fatalf("client sent command %v, want %v", cmd, "FETCH 1:4 (FLAGS)")
	}

	s.WriteString("* 2 FETCH (FLAGS (\\Seen))\r\n")
	s.WriteString("* 123 FETCH (FLAGS (\\Deleted))\r\n")
	s.WriteString("* 4 FETCH (FLAGS (\\Seen))\r\n")
	s.WriteString(tag + " OK FETCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Fetch() = %v", err)
	}

	msg := <-messages
	if msg.SeqNum != 2 {
		t.Errorf("First message has bad sequence number: %v", msg.SeqNum)
	}
	msg = <-messages
	if msg.SeqNum != 4 {
		t.Errorf("Second message has bad sequence number: %v", msg.SeqNum)
	}

	_, ok := <-messages
	if ok {
		t.Errorf("More than two messages")
	}
}

func TestClient_Fetch_Unilateral_Uid(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("1:4")
	fields := []imap.FetchItem{imap.FetchFlags}

	done := make(chan error, 1)
	messages := make(chan *imap.Message, 3)
	go func() {
		done <- c.UidFetch(seqset, fields, messages)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "UID FETCH 1:4 (FLAGS)" {
		t.Fatalf("client sent command %v, want %v", cmd, "UID FETCH 1:4 (FLAGS)")
	}

	s.WriteString("* 23 FETCH (UID 2 FLAGS (\\Seen))\r\n")
	s.WriteString("* 123 FETCH (FLAGS (\\Deleted))\r\n")
	s.WriteString("* 49 FETCH (UID 4 FLAGS (\\Seen))\r\n")
	s.WriteString(tag + " OK FETCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Fetch() = %v", err)
	}

	msg := <-messages
	if msg.Uid != 2 {
		t.Errorf("First message has bad UID: %v", msg.Uid)
	}
	msg = <-messages
	if msg.Uid != 4 {
		t.Errorf("Second message has bad UID: %v", msg.Uid)
	}

	_, ok := <-messages
	if ok {
		t.Errorf("More than two messages")
	}
}

func TestClient_Fetch_Uid_Dynamic(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("4:*")
	fields := []imap.FetchItem{imap.FetchFlags}

	done := make(chan error, 1)
	messages := make(chan *imap.Message, 1)
	go func() {
		done <- c.UidFetch(seqset, fields, messages)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "UID FETCH 4:* (FLAGS)" {
		t.Fatalf("client sent command %v, want %v", cmd, "UID FETCH 4:* (FLAGS)")
	}

	s.WriteString("* 23 FETCH (UID 2 FLAGS (\\Seen))\r\n")
	s.WriteString(tag + " OK FETCH completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Fetch() = %v", err)
	}

	msg, ok := <-messages
	if !ok {
		t.Errorf("No message supplied")
	} else if msg.Uid != 2 {
		t.Errorf("First message has bad UID: %v", msg.Uid)
	}
}

func TestClient_Store(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("2")

	done := make(chan error, 1)
	updates := make(chan *imap.Message, 1)
	go func() {
		done <- c.Store(seqset, imap.AddFlags, []interface{}{imap.SeenFlag, "foobar"}, updates)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "STORE 2 +FLAGS (\\Seen foobar)" {
		t.Fatalf("client sent command %v, want %v", cmd, "STORE 2 +FLAGS (\\Seen foobar)")
	}

	s.WriteString("* 2 FETCH (FLAGS (\\Seen foobar))\r\n")
	s.WriteString(tag + " OK STORE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Store() = %v", err)
	}

	msg := <-updates
	if len(msg.Flags) != 2 || msg.Flags[0] != "\\Seen" || msg.Flags[1] != "foobar" {
		t.Errorf("Bad message flags: %v", msg.Flags)
	}
}

func TestClient_Store_Silent(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("2:3")

	done := make(chan error, 1)
	go func() {
		done <- c.Store(seqset, imap.AddFlags, []interface{}{imap.SeenFlag, "foobar"}, nil)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "STORE 2:3 +FLAGS.SILENT (\\Seen foobar)" {
		t.Fatalf("client sent command %v, want %v", cmd, "STORE 2:3 +FLAGS.SILENT (\\Seen foobar)")
	}

	s.WriteString(tag + " OK STORE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Store() = %v", err)
	}
}

func TestClient_Store_Uid(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("27:901")

	done := make(chan error, 1)
	go func() {
		done <- c.UidStore(seqset, imap.AddFlags, []interface{}{imap.DeletedFlag, "foobar"}, nil)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "UID STORE 27:901 +FLAGS.SILENT (\\Deleted foobar)" {
		t.Fatalf("client sent command %v, want %v", cmd, "UID STORE 27:901 +FLAGS.SILENT (\\Deleted foobar)")
	}

	s.WriteString(tag + " OK STORE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.UidStore() = %v", err)
	}
}

func TestClient_Copy(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("2:4")

	done := make(chan error, 1)
	go func() {
		done <- c.Copy(seqset, "Sent")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "COPY 2:4 \"Sent\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "COPY 2:4 \"Sent\"")
	}

	s.WriteString(tag + " OK COPY completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Copy() = %v", err)
	}
}

func TestClient_Copy_Uid(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	seqset, _ := imap.ParseSeqSet("78:102")

	done := make(chan error, 1)
	go func() {
		done <- c.UidCopy(seqset, "Drafts")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "UID COPY 78:102 \"Drafts\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "UID COPY 78:102 \"Drafts\"")
	}

	s.WriteString(tag + " OK UID COPY completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.UidCopy() = %v", err)
	}
}

func TestClient_Unselect(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.SelectedState, nil)

	done := make(chan error, 1)
	go func() {
		done <- c.Unselect()
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "UNSELECT" {
		t.Fatalf("client sent command %v, want %v", cmd, "UNSELECT")
	}

	s.WriteString(tag + " OK UNSELECT completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Unselect() = %v", err)
	}

	if c.State() != imap.AuthenticatedState {
		t.Fatal("Client is not Authenticated after UNSELECT")
	}
}
