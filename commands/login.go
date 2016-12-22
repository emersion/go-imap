package commands

import (
	"errors"

	"github.com/emersion/go-imap"
)

// Login is a LOGIN command, as defined in RFC 3501 section 6.2.2.
type Login struct {
	Username string
	Password string
}

func (cmd *Login) Command() *imap.Command {
	return &imap.Command{
		Name:      imap.Login,
		Arguments: []interface{}{cmd.Username, cmd.Password},
	}
}

func (cmd *Login) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("Not enough arguments")
	}

	var ok bool
	if cmd.Username, ok = fields[0].(string); !ok {
		return errors.New("Username is not a string")
	}
	if cmd.Password, ok = fields[1].(string); !ok {
		return errors.New("Password is not a string")
	}

	return nil
}
