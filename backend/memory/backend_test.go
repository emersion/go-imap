package memory

import (
	"testing"
)

func TestAddUser(t *testing.T) {
	backend := New()
	t.Run("existing user returns error", func(t *testing.T) {
		err := backend.AddUser(&User{username: "username", password: "test", mailboxes: map[string]*Mailbox{}})
		if err == nil {
			t.Error("expected error, but got none")
		}
	})
	t.Run("new user is added to the list of users", func(t *testing.T) {
		err := backend.AddUser(&User{
			username:  "my_user",
			password:  "test",
			mailboxes: map[string]*Mailbox{},
		})
		if err != nil {
			t.Errorf("got error %s, but expected none", err)
		}
		if _, ok := backend.Users()["my_user"]; !ok {
			t.Errorf("user %s not added to list of users", "my_user")
		}
	})
}
