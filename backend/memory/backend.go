// A memory backend.
package memory

import (
	"errors"
	"time"

	"github.com/emersion/imap/backend"
	"github.com/emersion/imap/common"
)

type Backend struct {}

func (bkd *Backend) Login(username, password string) (backend.User, error) {
	if username != "username" || password != "password" {
		return nil, errors.New("Bad username or password")
	}

	user := &User{username: username}

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

	return user, nil
}

func New() *Backend {
	return &Backend{}
}
