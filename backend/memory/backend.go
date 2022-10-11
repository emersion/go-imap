// A memory backend.
package memory

import (
	"errors"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"sync"
)

type Backend struct {
	users    map[string]*User
	userLock sync.Mutex
}

func (be *Backend) Login(_ *imap.ConnInfo, username, password string) (backend.User, error) {
	user, ok := be.users[username]
	if ok && user.password == password {
		return user, nil
	}

	return nil, errors.New("Bad username or password")
}

func (be *Backend) AddUser(username, password string) (*User, error) {
	if _, ok := be.users[username]; ok {
		return nil, fmt.Errorf("user %s already exists", username)
	}

	user := &User{
		username: username,
		password: password,
	}

	be.userLock.Lock()
	defer be.userLock.Unlock()
	be.users[username] = user
	return user, nil
}

func (u *User) SetMailboxes(mailboxes map[string]*Mailbox) {
	u.mailboxes = mailboxes
}

func (be *Backend) GetUser(username string) (*User, error) {
	if user, ok := be.users[username]; ok {
		return user, nil
	}
	return nil, fmt.Errorf("no user exists with username %s", username)
}

func New() *Backend {
	return &Backend{
		users:    map[string]*User{},
		userLock: sync.Mutex{},
	}
}
