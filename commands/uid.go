package commands

import (
	"errors"
	"strings"

	"github.com/emersion/go-imap"
)

// A UID command.
// See RFC 3501 section 6.4.8
type Uid struct {
	Cmd imap.Commander
}

func (cmd *Uid) Command() *imap.Command {
	inner := cmd.Cmd.Command()

	args := []interface{}{inner.Name}
	args = append(args, inner.Arguments...)

	return &imap.Command{
		Name: imap.Uid,
		Arguments: args,
	}
}

func (cmd *Uid) Parse(fields []interface{}) error {
	if len(fields) < 0 {
		return errors.New("No command name specified")
	}

	name, ok := fields[0].(string)
	if !ok {
		return errors.New("Command name must be a string")
	}

	cmd.Cmd = &imap.Command{
		Name: strings.ToUpper(name), // Command names are case-insensitive
		Arguments: fields[1:],
	}

	return nil
}
