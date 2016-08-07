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

// A value that can be converted to a command.
type Commander interface {
	Command() *Command
}

// A command.
type Command struct {
	// The command tag. It acts as a unique identifier for this command. If empty,
	// the command is untagged.
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

func (cmd *Command) WriteTo(w Writer) error {
	ww := w.writer()

	tag := cmd.Tag
	if tag == "" {
		tag = "*"
	}

	fields := []interface{}{tag, cmd.Name}
	fields = append(fields, cmd.Arguments...)
	return ww.writeLine(fields...)
}

// Parse a command from fields.
func (cmd *Command) Parse(fields []interface{}) error {
	if len(fields) < 2 {
		return errors.New("imap: cannot parse command: no enough fields")
	}

	var ok bool
	if cmd.Tag, ok = fields[0].(string); !ok {
		return errors.New("imap: cannot parse command: invalid tag")
	}
	if cmd.Name, ok = fields[1].(string); !ok {
		return errors.New("imap: cannot parse command: invalid name")
	}
	cmd.Name = strings.ToUpper(cmd.Name) // Command names are case-insensitive

	cmd.Arguments = fields[2:]
	return nil
}
