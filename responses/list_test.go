package responses

import (
	"bytes"
	"testing"

	"github.com/emersion/go-imap"
)

func TestListSlashDelimiter(t *testing.T) {
	mbox := &imap.MailboxInfo{}

	if err := mbox.Parse([]interface{}{
		[]interface{}{"\\Unseen"},
		"/",
		"INBOX",
	}); err != nil {
		t.Error(err)
		t.FailNow()
	}

	if response := getListResponse(t, mbox); response != `* LIST (\Unseen) "/" INBOX`+"\r\n" {
		t.Error("Unexpected response:", response)
	}
}

func TestListNILDelimiter(t *testing.T) {
	mbox := &imap.MailboxInfo{}

	if err := mbox.Parse([]interface{}{
		[]interface{}{"\\Unseen"},
		nil,
		"INBOX",
	}); err != nil {
		t.Error(err)
		t.FailNow()
	}

	if response := getListResponse(t, mbox); response != `* LIST (\Unseen) NIL INBOX`+"\r\n" {
		t.Error("Unexpected response:", response)
	}
}

func newListResponse(mbox *imap.MailboxInfo) (l *List) {
	l = &List{Mailboxes: make(chan *imap.MailboxInfo)}

	go func() {
		l.Mailboxes <- mbox
		close(l.Mailboxes)
	}()

	return
}

func getListResponse(t *testing.T, mbox *imap.MailboxInfo) string {
	b := &bytes.Buffer{}
	w := imap.NewWriter(b)

	if err := newListResponse(mbox).WriteTo(w); err != nil {
		t.Error(err)
		t.FailNow()
	}

	return b.String()
}
