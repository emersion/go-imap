package commands

import (
	"errors"

	"github.com/emersion/go-imap"
)

// A COMPRESS command.
type Compress struct {
	// Name of the compression mechanism.
	Mechanism string
}

func (cmd *Compress) Command() *imap.Command {
	return &imap.Command{
		Name:      "COMPRESS",
		Arguments: []interface{}{cmd.Mechanism},
	}
}

func (cmd *Compress) Parse(fields []interface{}) (err error) {
	if len(fields) < 1 {
		return errors.New("No enough arguments")
	}

	var ok bool
	if cmd.Mechanism, ok = fields[0].(string); !ok {
		return errors.New("Compression mechanism name must be a string")
	}

	return nil
}
