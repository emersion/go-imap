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

// Implements the Commander interface.
func (cmd *Command) Command() *Command {
	return cmd
}

func (cmd *Command) WriteTo(w *Writer) (N int64, err error) {
	n, err := w.writeString(cmd.Tag + string(sp) + cmd.Name)
	N += int64(n)
	if err != nil {
		return
	}

	if len(cmd.Arguments) > 0 {
		n, err = w.WriteSp()
		N += int64(n)
		if err != nil {
			return
		}

		n, err = w.WriteFields(cmd.Arguments)
		N += int64(n)
		if err != nil {
			return
		}
	}

	n, err = w.WriteCrlf()
	N += int64(n)
	return
}

// Parse a command from fields.
func (cmd *Command) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("Cannot parse command")
	}

	var ok bool
	if cmd.Tag, ok = fields[0].(string); !ok {
		return errors.New("Cannot parse command tag")
	}
	if cmd.Name, ok = fields[1].(string); !ok {
		return errors.New("Cannot parse command name")
	}

	// Command names are case-insensitive
	cmd.Name = strings.ToUpper(cmd.Name)

	cmd.Arguments = fields[2:]

	return nil
}

// A value that can be converted to a command.
type Commander interface {
	Command() *Command
}
