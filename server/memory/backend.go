// A memory backend.
package memory

import (
	"errors"
)

type Backend struct {}

func (bkd *Backend) Login(username, password string) error {
	if username != "username" && password != "password" {
		return errors.New("Bad username or password")
	}

	return nil
}

func New() *Backend {
	return &Backend{}
}
