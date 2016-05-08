package memory

import (
	"github.com/emersion/imap/backend"
)

type User struct {
	username string
}

func (u *User) GetMailbox(name string) (mbox backend.Mailbox, err error) {
	return
}

func (u *User) ListMailboxes() ([]backend.Mailbox, error) {
	return nil, nil
}
