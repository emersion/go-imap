// A memory backend.
package memory

import (
	"errors"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
)

type Backend struct {
	users map[string]*User
}

func (bkd *Backend) Login(username, password string) (backend.User, error) {
	user, ok := bkd.users[username]
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

	now := time.Now()
	user.mailboxes = map[string]*Mailbox{
		"INBOX": &Mailbox{
			name: "INBOX",
			messages: []*Message{
				&Message{&imap.Message{
					Uid:   6,
					Flags: []string{"\\Seen"},
					Envelope: &imap.Envelope{
						Date:    now,
						Subject: "Hello World!",
						From:    []*imap.Address{},
						Sender:  []*imap.Address{},
						To:      []*imap.Address{},
					},
					BodyStructure: &imap.BodyStructure{MimeType: "text", MimeSubType: "plain"},
					Size:          uint32(len(body)),
					InternalDate:  now,
				}, []byte(body)},
			},
			user: user,
		},
	}

	return &Backend{
		users: map[string]*User{user.username: user},
	}
}
