package commands

import (
	"github.com/emersion/go-imap"
)

// An ENABLE command, defined in RFC 5161 section 3.1.
type Enable struct {
	Caps []string
}

func (cmd *Enable) Command() *imap.Command {
	args := make([]interface{}, len(cmd.Caps))
	for i, c := range cmd.Caps {
		args[i] = imap.RawString(c)
	}

	return &imap.Command{
		Name:      "ENABLE",
		Arguments: args,
	}
}

func (cmd *Enable) Parse(fields []interface{}) error {
	var err error
	cmd.Caps, err = imap.ParseStringList(fields)
	return err
}
