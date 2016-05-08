// A memory backend.
package memory

import (
	"errors"

	"github.com/emersion/imap/backend"
)

type Backend struct {}

func (bkd *Backend) Login(username, password string) (user backend.User, err error) {
	if username != "username" && password != "password" {
		err = errors.New("Bad username or password")
		return
	}

	user = &User{
		username: username,
		mailboxes: map[string]backend.Mailbox{
			"INBOX": &Mailbox{
				name: "INBOX",
				messages: []*Message{
					&Message{uid: 1},
					&Message{uid: 2},
					&Message{uid: 3},
				},
			},
		},
	}
	return
}

func New() *Backend {
	return &Backend{}
}
