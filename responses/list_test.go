package responses

import (
	"bytes"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSlashDelimiter(t *testing.T) {
	mbox := &imap.MailboxInfo{}

	require.NoError(t, mbox.Parse([]interface{}{
		[]interface{}{"\\Unseen"},
		"/",
		"INBOX",
	}))

	assert.Equal(t, `* LIST (\Unseen) "/" INBOX`+"\r\n", getListResponse(t, mbox))
}

func TestListNILDelimiter(t *testing.T) {
	mbox := &imap.MailboxInfo{}

	require.NoError(t, mbox.Parse([]interface{}{
		[]interface{}{"\\Unseen"},
		nil,
		"INBOX",
	}))

	assert.Equal(t, `* LIST (\Unseen) NIL INBOX`+"\r\n", getListResponse(t, mbox))
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

	require.NoError(t, newListResponse(mbox).WriteTo(w))

	return b.String()
}
