// A memory backend.
package memory

import (
	"errors"
	"time"

	"github.com/emersion/imap/backend"
	"github.com/emersion/imap/common"
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

	now := time.Now()
	user.mailboxes = map[string]*Mailbox{
		"INBOX": &Mailbox{
			name: "INBOX",
			messages: []*Message{
				&Message{&common.Message{
					Uid: 6,
					Envelope: &common.Envelope{Subject: "Hello World!"},
					BodyStructure: &common.BodyStructure{MimeType: "text", MimeSubType: "plain"},
					Body: map[string]*common.Literal{"BODY[]": common.NewLiteral([]byte("Hi there! How are you?"))},
					Size: 22,
					InternalDate: &now,
				}},
			},
			user: user,
		},
	}

	return &Backend{
		users: map[string]*User{user.username: user},
	}
}
