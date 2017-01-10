// A memory backend.
package memory

import (
	"errors"
	"time"

	"github.com/emersion/go-imap/backend"
)

type Backend struct {
	users map[string]*User
}

func (be *Backend) Login(username, password string) (backend.User, error) {
	user, ok := be.users[username]
	if ok && user.password == password {
		return user, nil
	}

	return nil, errors.New("Bad username or password")
}

func New() *Backend {
	user := &User{username: "username", password: "password"}

	body := `From: contact@example.org
To: contact@example.org
Subject: A little message, just for you
Date: Wed, 11 May 2016 14:31:59 +0000
Message-ID: <0000000@localhost/>
Content-Type: text/plain

Hi there :)`

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
