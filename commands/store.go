package commands

import (
	"errors"
	"strings"

	"github.com/emersion/go-imap"
)

// Store is a STORE command, as defined in RFC 3501 section 6.4.6.
type Store struct {
	SeqSet *imap.SeqSet
	Item   string
	Value  interface{}
}

func (cmd *Store) Command() *imap.Command {
	return &imap.Command{
		Name:      imap.Store,
		Arguments: []interface{}{cmd.SeqSet, cmd.Item, cmd.Value},
	}
}

func (cmd *Store) Parse(fields []interface{}) (err error) {
	if len(fields) < 3 {
		return errors.New("No enough arguments")
	}

	seqset, ok := fields[0].(string)
	if !ok {
		return errors.New("Invalid sequence set")
	}
	if cmd.SeqSet, err = imap.ParseSeqSet(seqset); err != nil {
		return err
	}

	if cmd.Item, ok = fields[1].(string); !ok {
		return errors.New("Item name must be a string")
	}
	cmd.Item = strings.ToUpper(cmd.Item)

	cmd.Value = fields[2]

	return
}
