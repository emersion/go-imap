package commands

import (
	"errors"
	"strings"

	"github.com/emersion/go-imap"
)

// UID is a UID command, as defined in RFC 3501 section 6.4.8. It wraps another
// command (e.g. wrapping a Fetch command will result in a UID FETCH).
type UID struct {
	Cmd imap.Commander
}

func (cmd *UID) Command() *imap.Command {
	inner := cmd.Cmd.Command()

	args := []interface{}{imap.RawString(inner.Name)}
	args = append(args, inner.Arguments...)

	return &imap.Command{
		Name:      "UID",
		Arguments: args,
	}
}

func (cmd *UID) Parse(fields []interface{}) error {
	if len(fields) < 0 {
		return errors.New("no command name specified")
	}

	name, ok := fields[0].(string)
	if !ok {
		return errors.New("command name must be a string")
	}

	cmd.Cmd = &imap.Command{
		Name:      strings.ToUpper(name), // Command names are case-insensitive
		Arguments: fields[1:],
	}

	return nil
}
