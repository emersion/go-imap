package common

import (
	"io"
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

// Implements io.WriterTo interface.
func (c *Command) WriteTo(w io.Writer) (N int64, err error) {
	n, err := w.Write([]byte(c.Tag + " " + c.Name))
	if err != nil {
		return
	}
	N += int64(n)

	var literals []*Literal
	if len(c.Arguments) > 0 {
		var args string
		args, err = formatFields(c.Arguments)
		if err != nil {
			return
		}

		n, err = w.Write([]byte(" " + args))
		if err != nil {
			return
		}
		N += int64(n)

		for _, f := range c.Arguments {
			if literal, ok := f.(*Literal); ok {
				literals = append(literals, literal)
			}
		}
	}

	n, err = w.Write([]byte("\n"))
	if err != nil {
		return
	}
	N += int64(n)

	if len(literals) > 0 {
		// TODO: send literals
	}

	return
}

// A value that can be converted to a command.
type Commander interface {
	Command() *Command
}
