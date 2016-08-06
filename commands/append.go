package commands

import (
	"errors"
	"time"

	imap "github.com/emersion/go-imap/common"
	"github.com/emersion/go-imap/utf7"
)

// An APPEND command.
// See RFC 3501 section 6.3.11
type Append struct {
	Mailbox string
	Flags []string
	Date time.Time
	Message *imap.Literal
}

func (cmd *Append) Command() *imap.Command {
	var args []interface{}

	mailbox, _ := utf7.Encoder.String(cmd.Mailbox)
	args = append(args, mailbox)

	if cmd.Flags != nil {
		flags := make([]interface{}, len(cmd.Flags))
		for i, flag := range cmd.Flags {
			flags[i] = flag
		}
		args = append(args, flags)
	}

	if !cmd.Date.IsZero() {
		args = append(args, cmd.Date)
	}

	args = append(args, cmd.Message)

	return &imap.Command{
		Name: imap.Append,
		Arguments: args,
	}
}

func (cmd *Append) Parse(fields []interface{}) (err error) {
	if len(fields) < 2 {
		return errors.New("No enough arguments")
	}

	// Parse mailbox name
	mailbox, ok := fields[0].(string)
	if !ok {
		return errors.New("Mailbox name must be a string")
	}
	if cmd.Mailbox, err = utf7.Decoder.String(mailbox); err != nil {
		return err
	}

	// Parse message literal
	litIndex := len(fields) - 1
	cmd.Message, ok = fields[litIndex].(*imap.Literal)
	if !ok {
		return errors.New("Message must be a literal")
	}

	// Remaining fields a optional
	fields = fields[1:litIndex]
	if len(fields) > 0 {
		// Parse flags list
		if flags, ok := fields[0].([]interface{}); ok {
			if cmd.Flags, err = imap.ParseStringList(flags); err != nil {
				return err
			}
			fields = fields[1:]
		}

		// Parse date
		if len(fields) > 0 {
			date, ok := fields[0].(string)
			if !ok {
				return errors.New("Date must be a string")
			}
			if cmd.Date, err = imap.ParseDateTime(date); err != nil {
				return err
			}
		}
	}

	return
}
