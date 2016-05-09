package common

import (
	"errors"
)

// Basic mailbox info.
type MailboxInfo struct {
	// The mailbox flags.
	Flags []string
	// The server's path separator.
	Delimiter string
	// The mailbox name.
	Name string
}

// Parse mailbox info from fields.
func (info *MailboxInfo) Parse(fields []interface{}) error {
	if len(fields) < 3 {
		return errors.New("Mailbox info needs at least 3 fields")
	}

	flags, _ := fields[0].([]interface{})
	info.Flags, _ = ParseStringList(flags)

	info.Delimiter, _ = fields[1].(string)
	info.Name, _ = fields[2].(string)

	return nil
}

// Format mailbox info to fields.
func (info *MailboxInfo) Format() []interface{} {
	return []interface{}{FormatStringList(info.Flags), info.Delimiter, info.Name}
}

// A mailbox status.
type MailboxStatus struct {
	// The mailbox name.
	Name string
	// True if the mailbox is open in read-only mode.
	ReadOnly bool
	// The mailbox flags.
	Flags []string
	// The mailbox permanent flags.
	PermanentFlags []string
	// The mailbox items that are currently filled in.
	Items []string

	// The number of messages in this mailbox.
	Messages uint32
	// The number of messages not seen since the last time the mailbox was opened.
	Recent uint32
	// The number of unread messages.
	Unseen uint32
	// The next UID.
	UidNext uint32
	// Together with a UID, it is a unique identifier for a message.
	UidValidity uint32
}
