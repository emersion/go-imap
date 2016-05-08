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
			"INBOX": &Mailbox{},
		},
	}
	return
}

func New() *Backend {
	return &Backend{}
}
