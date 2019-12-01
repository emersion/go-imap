package client

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

func TestClient_Select(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	var mbox *imap.MailboxStatus
	done := make(chan error, 1)
	go func() {
		var err error
		mbox, err = c.Select("INBOX", false)
		done <- err
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "SELECT INBOX" {
		t.Fatalf("client sent command %v, want SELECT \"INBOX\"", cmd)
	}

	s.WriteString("* 172 EXISTS\r\n")
	s.WriteString("* 1 RECENT\r\n")
	s.WriteString("* OK [UNSEEN 12] Message 12 is first unseen\r\n")
	s.WriteString("* OK [UIDVALIDITY 3857529045] UIDs valid\r\n")
	s.WriteString("* OK [UIDNEXT 4392] Predicted next UID\r\n")
	s.WriteString("* FLAGS (\\Answered \\Flagged \\Deleted \\Seen \\Draft)\r\n")
	s.WriteString("* OK [PERMANENTFLAGS (\\Deleted \\Seen \\*)] Limited\r\n")
	s.WriteString(tag + " OK SELECT completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Select() = %v", err)
	}

	want := &imap.MailboxStatus{
		Name:           "INBOX",
		ReadOnly:       false,
		Flags:          []string{imap.AnsweredFlag, imap.FlaggedFlag, imap.DeletedFlag, imap.SeenFlag, imap.DraftFlag},
		PermanentFlags: []string{imap.DeletedFlag, imap.SeenFlag, "\\*"},
		UnseenSeqNum:   12,
		Messages:       172,
		Recent:         1,
		UidNext:        4392,
		UidValidity:    3857529045,
	}
	mbox.Items = nil
	if !reflect.DeepEqual(mbox, want) {
		t.Errorf("c.Select() = \n%+v\n want \n%+v", mbox, want)
	}
}

func TestClient_Select_ReadOnly(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	var mbox *imap.MailboxStatus
	done := make(chan error, 1)
	go func() {
		var err error
		mbox, err = c.Select("INBOX", true)
		done <- err
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "EXAMINE INBOX" {
		t.Fatalf("client sent command %v, want EXAMINE \"INBOX\"", cmd)
	}

	s.WriteString(tag + " OK [READ-ONLY] EXAMINE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Select() = %v", err)
	}

	if !mbox.ReadOnly {
		t.Errorf("c.Select().ReadOnly = false, want true")
	}
}

func TestClient_Create(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	done := make(chan error, 1)
	go func() {
		done <- c.Create("New Mailbox")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "CREATE \"New Mailbox\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "CREATE \"New Mailbox\"")
	}

	s.WriteString(tag + " OK CREATE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Create() = %v", err)
	}
}

func TestClient_Delete(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	done := make(chan error, 1)
	go func() {
		done <- c.Delete("Old Mailbox")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "DELETE \"Old Mailbox\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "DELETE \"Old Mailbox\"")
	}

	s.WriteString(tag + " OK DELETE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Delete() = %v", err)
	}
}

func TestClient_Rename(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	done := make(chan error, 1)
	go func() {
		done <- c.Rename("Old Mailbox", "New Mailbox")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "RENAME \"Old Mailbox\" \"New Mailbox\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "RENAME \"Old Mailbox\" \"New Mailbox\"")
	}

	s.WriteString(tag + " OK RENAME completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Rename() = %v", err)
	}
}

func TestClient_Subscribe(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	done := make(chan error, 1)
	go func() {
		done <- c.Subscribe("Mailbox")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "SUBSCRIBE \"Mailbox\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "SUBSCRIBE \"Mailbox\"")
	}

	s.WriteString(tag + " OK SUBSCRIBE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Subscribe() = %v", err)
	}
}

func TestClient_Unsubscribe(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	done := make(chan error, 1)
	go func() {
		done <- c.Unsubscribe("Mailbox")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "UNSUBSCRIBE \"Mailbox\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "UNSUBSCRIBE \"Mailbox\"")
	}

	s.WriteString(tag + " OK UNSUBSCRIBE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Unsubscribe() = %v", err)
	}
}

func TestClient_List(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	done := make(chan error, 1)
	mailboxes := make(chan *imap.MailboxInfo, 3)
	go func() {
		done <- c.List("", "%", mailboxes)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "LIST \"\" \"%\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "LIST \"\" \"%\"")
	}

	s.WriteString("* LIST (flag1) \"/\" INBOX\r\n")
	s.WriteString("* LIST (flag2 flag3) \"/\" Drafts\r\n")
	s.WriteString("* LIST () \"/\" Sent\r\n")
	s.WriteString(tag + " OK LIST completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.List() = %v", err)
	}

	want := []struct {
		name       string
		attributes []string
	}{
		{"INBOX", []string{"flag1"}},
		{"Drafts", []string{"flag2", "flag3"}},
		{"Sent", []string{}},
	}

	i := 0
	for mbox := range mailboxes {
		if mbox.Name != want[i].name {
			t.Errorf("Bad mailbox name for %v: %v, want %v", i, mbox.Name, want[i].name)
		}

		if !reflect.DeepEqual(mbox.Attributes, want[i].attributes) {
			t.Errorf("Bad mailbox attributes for %v: %v, want %v", i, mbox.Attributes, want[i].attributes)
		}

		i++
	}
}

func TestClient_Lsub(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	done := make(chan error, 1)
	mailboxes := make(chan *imap.MailboxInfo, 1)
	go func() {
		done <- c.Lsub("", "%", mailboxes)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "LSUB \"\" \"%\"" {
		t.Fatalf("client sent command %v, want %v", cmd, "LSUB \"\" \"%\"")
	}

	s.WriteString("* LSUB () \"/\" INBOX\r\n")
	s.WriteString(tag + " OK LSUB completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Lsub() = %v", err)
	}

	mbox := <-mailboxes
	if mbox.Name != "INBOX" {
		t.Errorf("Bad mailbox name: %v", mbox.Name)
	}
	if len(mbox.Attributes) != 0 {
		t.Errorf("Bad mailbox flags: %v", mbox.Attributes)
	}
}

func TestClient_Status(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	done := make(chan error, 1)
	var mbox *imap.MailboxStatus
	go func() {
		var err error
		mbox, err = c.Status("INBOX", []imap.StatusItem{imap.StatusMessages, imap.StatusRecent})
		done <- err
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "STATUS INBOX (MESSAGES RECENT)" {
		t.Fatalf("client sent command %v, want %v", cmd, "STATUS \"INBOX\" (MESSAGES RECENT)")
	}

	s.WriteString("* STATUS INBOX (MESSAGES 42 RECENT 1)\r\n")
	s.WriteString(tag + " OK STATUS completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Status() = %v", err)
	}

	if mbox.Messages != 42 {
		t.Errorf("Bad mailbox messages: %v", mbox.Messages)
	}
	if mbox.Recent != 1 {
		t.Errorf("Bad mailbox recent: %v", mbox.Recent)
	}
}

type literalWrap struct {
	io.Reader
	L int
}

func (lw literalWrap) Len() int {
	return lw.L
}

func TestClient_Append_SmallerLiteral(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	msg := "Hello World!\r\nHello Gophers!\r\n"
	date := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	flags := []string{imap.SeenFlag, imap.DraftFlag}

	r := bytes.NewBufferString(msg)

	done := make(chan error, 1)
	go func() {
		done <- c.Append("INBOX", flags, date, literalWrap{r, 35})

		// The buffer is not flushed on error, force it so io.ReadFull can
		// continue.
		c.conn.Flush()
	}()

	tag, _ := s.ScanCmd()
	s.WriteString("+ send literal\r\n")

	b := make([]byte, 30)
	// The client will close connection.
	if _, err := io.ReadFull(s, b); err != io.EOF {
		t.Error("Expected EOF, got", err)
	}

	s.WriteString(tag + " OK APPEND completed\r\n")

	err, ok := (<-done).(imap.LiteralLengthErr)
	if !ok {
		t.Fatalf("c.Append() = %v", err)
	}
	if err.Expected != 35 {
		t.Fatalf("err.Expected = %v", err.Expected)
	}
	if err.Actual != 30 {
		t.Fatalf("err.Actual = %v", err.Actual)
	}
}

func TestClient_Append_BiggerLiteral(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	msg := "Hello World!\r\nHello Gophers!\r\n"
	date := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	flags := []string{imap.SeenFlag, imap.DraftFlag}

	r := bytes.NewBufferString(msg)

	done := make(chan error, 1)
	go func() {
		done <- c.Append("INBOX", flags, date, literalWrap{r, 25})

		// The buffer is not flushed on error, force it so io.ReadFull can
		// continue.
		c.conn.Flush()
	}()

	tag, _ := s.ScanCmd()
	s.WriteString("+ send literal\r\n")

	// The client will close connection.
	b := make([]byte, 25)
	if _, err := io.ReadFull(s, b); err != io.EOF {
		t.Error("Expected EOF, got", err)
	}

	s.WriteString(tag + " OK APPEND completed\r\n")

	err, ok := (<-done).(imap.LiteralLengthErr)
	if !ok {
		t.Fatalf("c.Append() = %v", err)
	}
	if err.Expected != 25 {
		t.Fatalf("err.Expected = %v", err.Expected)
	}
	if err.Actual != 30 {
		t.Fatalf("err.Actual = %v", err.Actual)
	}
}

func TestClient_Append(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	msg := "Hello World!\r\nHello Gophers!\r\n"
	date := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	flags := []string{imap.SeenFlag, imap.DraftFlag}

	done := make(chan error, 1)
	go func() {
		done <- c.Append("INBOX", flags, date, bytes.NewBufferString(msg))
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "APPEND INBOX (\\Seen \\Draft) \"10-Nov-2009 23:00:00 +0000\" {30}" {
		t.Fatalf("client sent command %v, want %v", cmd, "APPEND \"INBOX\" (\\Seen \\Draft) \"10-Nov-2009 23:00:00 +0000\" {30}")
	}

	s.WriteString("+ send literal\r\n")

	b := make([]byte, 30)
	if _, err := io.ReadFull(s, b); err != nil {
		t.Fatal(err)
	} else if string(b) != msg {
		t.Fatal("Bad literal:", string(b))
	}

	s.WriteString(tag + " OK APPEND completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Append() = %v", err)
	}
}

func TestClient_Append_failed(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	setClientState(c, imap.AuthenticatedState, nil)

	// First the server refuses

	msg := "First try"
	done := make(chan error, 1)
	go func() {
		done <- c.Append("INBOX", nil, time.Time{}, bytes.NewBufferString(msg))
	}()

	tag, _ := s.ScanCmd()
	s.WriteString(tag + " BAD APPEND failed\r\n")

	if err := <-done; err == nil {
		t.Fatal("c.Append() = nil, want an error from the server")
	}

	// Try a second time, the server accepts

	msg = "Second try"
	go func() {
		done <- c.Append("INBOX", nil, time.Time{}, bytes.NewBufferString(msg))
	}()

	tag, _ = s.ScanCmd()
	s.WriteString("+ send literal\r\n")

	b := make([]byte, len(msg))
	if _, err := io.ReadFull(s, b); err != nil {
		t.Fatal(err)
	} else if string(b) != msg {
		t.Fatal("Bad literal:", string(b))
	}

	s.WriteString(tag + " OK APPEND completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Append() = %v", err)
	}
}
