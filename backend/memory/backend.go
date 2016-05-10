// A memory backend.
package memory

import (
	"errors"
	"time"

	"github.com/emersion/imap/backend"
	"github.com/emersion/imap/common"
)

type Backend struct {}

func (bkd *Backend) Login(username, password string) (user backend.User, err error) {
	if username != "username" || password != "password" {
		err = errors.New("Bad username or password")
		return
	}

	now := time.Now()

	user = &User{
		username: username,
		mailboxes: map[string]*Mailbox{
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
			},
		},
	}
	return
}

func New() *Backend {
	return &Backend{}
}
