package common

import (
	"errors"
	"strings"
)

// IMAP4rev1 commands.
const (
	Capability string = "CAPABILITY"
	Noop = "NOOP"
	Logout = "LOGOUT"
	StartTLS = "STARTTLS"

	Authenticate = "AUTHENTICATE"
	Login = "LOGIN"

	Select = "SELECT"
	Examine = "EXAMINE"
	Create = "CREATE"
	Delete = "DELETE"
	Rename = "RENAME"
	Subscribe = "SUBSCRIBE"
	Unsubscribe = "UNSUBSCRIBE"
	List = "LIST"
	Lsub = "LSUB"
	Status = "STATUS"
	Append = "APPEND"

	Check = "CHECK"
	Close = "CLOSE"
	Expunge = "EXPUNGE"
	Search = "SEARCH"
	Fetch = "FETCH"
	Store = "STORE"
	Copy = "COPY"
	Uid = "UID"
)

// A command.
type Command struct {
	// The command tag. It acts as a unique identifier for this command.
	Tag string
	// The command name.
	Name string
	// The command arguments.
	Arguments []interface{}
}

func (c *Command) WriteTo(w *Writer) (N int64, err error) {
	n, err := w.writeString(c.Tag + string(sp) + c.Name)
	N += int64(n)
	if err != nil {
		return
	}

	if len(c.Arguments) > 0 {
		n, err = w.WriteSp()
		N += int64(n)
		if err != nil {
			return
		}

		n, err = w.WriteFields(c.Arguments)
		N += int64(n)
		if err != nil {
			return
		}
	}

	n, err = w.WriteCrlf()
	N += int64(n)
	return
}

func (c *Command) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("Cannot parse command")
	}

	var ok bool

	if c.Tag, ok = fields[0].(string); !ok {
		return errors.New("Cannot parse command tag")
	}

	if c.Name, ok = fields[1].(string); !ok {
		return errors.New("Cannot parse command name")
	}

	// Command names are case-insensitive
	c.Name = strings.ToUpper(c.Name)

	c.Arguments = fields[2:]

	return nil
}

// A value that can be converted to a command.
type Commander interface {
	Command() *Command
}
