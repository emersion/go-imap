package common

import (
	"errors"
	"io"
	"strconv"
)

const (
	Capability string = "CAPABILITY"
	Noop = "NOOP"
	Logout = "LOGOUT"
	Starttls = "STARTTLS"

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
	Tag string
	Name string
	Arguments []interface{}
}

// Implements io.WriterTo interface.
func (c *Command) WriteTo(w io.Writer) (N int64, err error) {
	n, err := w.Write([]byte(c.Tag + " " + c.Name))
	if err != nil {
		return
	}
	N += n

	if len(c.Arguments) > 0 {
		var args string
		args, err = formatList(c.Arguments)
		if err != nil {
			return
		}

		n, err = w.Write([]byte(" " + args))
		if err != nil {
			return
		}
		N += n
	}

	return
}

// A value that can be converted to a command.
type Commander interface {
	Command() *Command
}
