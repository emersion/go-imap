package memory

import (
	"errors"
	"io/ioutil"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
)

type User struct {
	username  string
	password  string
	mailboxes map[string]*Mailbox
}

func (u *User) Username() string {
	return u.username
}

func (u *User) ListMailboxes(subscribed bool) (info []imap.MailboxInfo, err error) {
	for _, mailbox := range u.mailboxes {
		if subscribed && !mailbox.Subscribed {
			continue
		}

		mboxInfo, err := mailbox.Info()
		if err != nil {
			return nil, err
		}
		info = append(info, *mboxInfo)
	}
	return
}

func (u *User) GetMailbox(name string, readOnly bool, conn backend.Conn) (*imap.MailboxStatus, backend.Mailbox, error) {
	mailbox, ok := u.mailboxes[name]
	if !ok {
		return nil, nil, backend.ErrNoSuchMailbox
	}

	status, err := u.Status(name, []imap.StatusItem{
		imap.StatusMessages, imap.StatusRecent, imap.StatusUnseen,
		imap.StatusUidNext, imap.StatusUidValidity,
	})
	if err != nil {
		return nil, nil, err
	}

	return status, &SelectedMailbox{
		Mailbox:  mailbox,
		conn:     conn,
		readOnly: readOnly,
	}, nil
}

func (u *User) Status(name string, items []imap.StatusItem) (*imap.MailboxStatus, error) {
	mbox, ok := u.mailboxes[name]
	if !ok {
		return nil, backend.ErrNoSuchMailbox
	}

	status := imap.NewMailboxStatus(mbox.name, items)
	status.Flags = mbox.flags()
	status.PermanentFlags = []string{"\\*"}
	status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = uint32(len(mbox.Messages))
		case imap.StatusUidNext:
			status.UidNext = mbox.uidNext()
		case imap.StatusUidValidity:
			status.UidValidity = 1
		case imap.StatusRecent:
			status.Recent = 0 // TODO
		case imap.StatusUnseen:
			status.Unseen = 0 // TODO
		}
	}

	return status, nil
}

func (u *User) SetSubscribed(name string, subscribed bool) error {
	mbox, ok := u.mailboxes[name]
	if !ok {
		return backend.ErrNoSuchMailbox
	}

	mbox.Subscribed = subscribed
	return nil
}

func (u *User) CreateMessage(mboxName string, flags []string, date time.Time, body imap.Literal, _ backend.Mailbox) error {
	mbox, ok := u.mailboxes[mboxName]
	if !ok {
		return backend.ErrNoSuchMailbox
	}

	if date.IsZero() {
		date = time.Now()
	}

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	mbox.Messages = append(mbox.Messages, &Message{
		Uid:   mbox.uidNext(),
		Date:  date,
		Size:  uint32(len(b)),
		Flags: flags,
		Body:  b,
	})
	return nil
}

func (u *User) CreateMailbox(name string) error {
	if _, ok := u.mailboxes[name]; ok {
		return backend.ErrMailboxAlreadyExists
	}

	u.mailboxes[name] = &Mailbox{name: name, user: u}
	return nil
}

func (u *User) DeleteMailbox(name string) error {
	if name == "INBOX" {
		return errors.New("Cannot delete INBOX")
	}
	if _, ok := u.mailboxes[name]; !ok {
		return backend.ErrNoSuchMailbox
	}

	delete(u.mailboxes, name)
	return nil
}

func (u *User) RenameMailbox(existingName, newName string) error {
	mbox, ok := u.mailboxes[existingName]
	if !ok {
		return backend.ErrNoSuchMailbox
	}

	u.mailboxes[newName] = &Mailbox{
		name:     newName,
		Messages: mbox.Messages,
		user:     u,
	}

	mbox.Messages = nil

	if existingName != "INBOX" {
		delete(u.mailboxes, existingName)
	}

	return nil
}

func (u *User) Logout() error {
	return nil
}
