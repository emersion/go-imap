// A memory backend.
package memory

import (
	"errors"
	"fmt"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
)

type Backend struct {
	users map[string]*User
}

func (be *Backend) Login(_ *imap.ConnInfo, username, password string) (backend.User, error) {
	user, ok := be.users[username]
	if ok && user.password == password {
		return user, nil
	}

	return nil, errors.New("Bad username or password")
}

func New() *Backend {
	user := &User{username: "username", password: "password"}

	body := "From: contact@example.org\r\n" +
		"To: contact@example.org\r\n" +
		"Subject: A little message, just for you\r\n" +
		"Date: Wed, 11 May 2016 14:31:59 +0000\r\n" +
		"Message-ID: <0000000@localhost/>\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Hi there :)"

	user.mailboxes = map[string]*Mailbox{
		"INBOX": {
			name: "INBOX",
			user: user,
			Messages: []*Message{
				{
					Uid:   6,
					Date:  time.Now(),
					Flags: []string{"\\Seen"},
					Size:  uint32(len(body)),
					Body:  []byte(body),
				},
			},
		},
	}

	return &Backend{
		users: map[string]*User{user.username: user},
	}
}

// NewUser adds a user to the backend.
func (be *Backend) NewUser(username, password string) (*User, error) {
	_, ok := be.users[username]
	if ok {
		return nil, fmt.Errorf("user %s is already defined.", username)
	}
	u := &User{username: username, password: password, mailboxes: make(map[string]*Mailbox)}
	be.users[username] = u
	return u, nil
}

// DeleteUser removes a user from the backend.
func (be *Backend) DeleteUser(username string) error {
	_, ok := be.users[username]
	if !ok {
		return fmt.Errorf("user %s is not defined.", username)
	}
	delete(be.users, username)
	return nil
}
