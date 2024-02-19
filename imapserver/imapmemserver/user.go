package imapmemserver

import (
	"crypto/subtle"
	"sort"
	"strings"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

const mailboxDelim rune = '/'

type User struct {
	username, password string

	mutex           sync.Mutex
	mailboxes       map[string]*Mailbox
	prevUidValidity uint32
}

func NewUser(username, password string) *User {
	return &User{
		username:  username,
		password:  password,
		mailboxes: make(map[string]*Mailbox),
	}
}

func (u *User) Login(username, password string) error {
	if username != u.username {
		return imapserver.ErrAuthFailed
	}
	if subtle.ConstantTimeCompare([]byte(password), []byte(u.password)) != 1 {
		return imapserver.ErrAuthFailed
	}
	return nil
}

func (u *User) mailboxLocked(name string) (*Mailbox, error) {
	mbox := u.mailboxes[name]
	if mbox == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeNonExistent,
			Text: "No such mailbox",
		}
	}
	return mbox, nil
}

func (u *User) mailbox(name string) (*Mailbox, error) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	return u.mailboxLocked(name)
}

func (u *User) Status(name string, options *imap.StatusOptions) (*imap.StatusData, error) {
	mbox, err := u.mailbox(name)
	if err != nil {
		return nil, err
	}
	return mbox.StatusData(options), nil
}

func (u *User) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	// TODO: fail if ref doesn't exist

	if len(patterns) == 0 {
		return w.WriteList(&imap.ListData{
			Attrs: []imap.MailboxAttr{imap.MailboxAttrNoSelect},
			Delim: mailboxDelim,
		})
	}

	var l []imap.ListData
	for name, mbox := range u.mailboxes {
		match := false
		for _, pattern := range patterns {
			match = imapserver.MatchList(name, mailboxDelim, ref, pattern)
			if match {
				break
			}
		}
		if !match {
			continue
		}

		data := mbox.list(options)
		if data != nil {
			l = append(l, *data)
		}
	}

	sort.Slice(l, func(i, j int) bool {
		return l[i].Mailbox < l[j].Mailbox
	})

	for _, data := range l {
		if err := w.WriteList(&data); err != nil {
			return err
		}
	}

	return nil
}

func (u *User) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	mbox, err := u.mailbox(mailbox)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeTryCreate,
			Text: "No such mailbox",
		}
	}
	return mbox.appendLiteral(r, options)
}

func (u *User) Create(name string, options *imap.CreateOptions) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	name = strings.TrimRight(name, string(mailboxDelim))

	if u.mailboxes[name] != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAlreadyExists,
			Text: "Mailbox already exists",
		}
	}

	// UIDVALIDITY must change if a mailbox is deleted and re-created with the
	// same name.
	u.prevUidValidity++
	u.mailboxes[name] = NewMailbox(name, u.prevUidValidity)
	return nil
}

func (u *User) Delete(name string) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if _, err := u.mailboxLocked(name); err != nil {
		return err
	}

	delete(u.mailboxes, name)
	return nil
}

func (u *User) Rename(oldName, newName string) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	newName = strings.TrimRight(newName, string(mailboxDelim))

	mbox, err := u.mailboxLocked(oldName)
	if err != nil {
		return err
	}

	if u.mailboxes[newName] != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAlreadyExists,
			Text: "Mailbox already exists",
		}
	}

	mbox.rename(newName)
	u.mailboxes[newName] = mbox
	delete(u.mailboxes, oldName)
	return nil
}

func (u *User) Subscribe(name string) error {
	mbox, err := u.mailbox(name)
	if err != nil {
		return err
	}
	mbox.SetSubscribed(true)
	return nil
}

func (u *User) Unsubscribe(name string) error {
	mbox, err := u.mailbox(name)
	if err != nil {
		return err
	}
	mbox.SetSubscribed(false)
	return nil
}

func (u *User) Namespace() (*imap.NamespaceData, error) {
	return &imap.NamespaceData{
		Personal: []imap.NamespaceDescriptor{{Delim: mailboxDelim}},
	}, nil
}
