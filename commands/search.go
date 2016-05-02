package commands

import (
	imap "github.com/emersion/imap/common"
)

// A SEARCH command.
// See RFC 3501 section 6.4.4
type Search struct {
	Charset string
	Criteria []interface{}
}

func (c *Search) Command() *imap.Command {
	var args []interface{}
	if c.Charset != "" {
		args = append(args, "CHARSET", c.Charset)
	}
	args = append(args, c.Criteria...)

	return &imap.Command{
		Name: imap.Search,
		Arguments: args,
	}
}
