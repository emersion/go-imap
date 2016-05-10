package memory

import (
	"errors"

	"github.com/emersion/imap/backend"
)

type User struct {
	username string
	mailboxes map[string]backend.Mailbox
}

func (u *User) ListMailboxes() (mailboxes []backend.Mailbox, err error) {
	for _, mailbox := range u.mailboxes {
		mailboxes = append(mailboxes, mailbox)
	}
	return
}

func (u *User) GetMailbox(name string) (mailbox backend.Mailbox, err error) {
	mailbox, ok := u.mailboxes[name]
	if !ok {
		err = errors.New("No such mailbox")
	}
	return
}

func (u *User) CreateMailbox(name string) error {
	if _, ok := u.mailboxes[name]; ok {
		return errors.New("Mailbox already exists")
	}

	u.mailboxes[name] = &Mailbox{name: name}
	return nil
}

func (u *User) DeleteMailbox(name string) error {
	if name == "INBOX" {
		return errors.New("Cannot delete INBOX")
	}
	if _, ok := u.mailboxes[name]; !ok {
		return errors.New("No such mailbox")
	}

	delete(u.mailboxes, name)
	return nil
}
