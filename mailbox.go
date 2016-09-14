package imap

import (
	"errors"
	"strings"

	"github.com/emersion/go-imap/utf7"
)

// The primary mailbox, as defined in RFC 3501 section 5.1.
const InboxName = "INBOX"

// Returns the canonical form of a mailbox name. Mailbox names can be
// case-sensitive or case-insensitive depending on the backend implementation.
// The spacial INBOX mailbox is case-insensitive.
func CanonicalMailboxName(name string) string {
	if strings.ToUpper(name) == InboxName {
		return InboxName
	}
	return name
}

// Mailbox attributes definied in RFC 3501 section 7.2.2.
const (
	// It is not possible for any child levels of hierarchy to exist under this\
	// name; no child levels exist now and none can be created in the future.
	NoInferiorsAttr = "\\Noinferiors"
	// It is not possible to use this name as a selectable mailbox.
	NoSelectAttr = "\\Noselect"
	// The mailbox has been marked "interesting" by the server; the mailbox
	// probably contains messages that have been added since the last time the
	// mailbox was selected.
	MarkedAttr = "\\Marked"
	// The mailbox does not contain any additional messages since the last time
	// the mailbox was selected.
	UnmarkedAttr = "\\Unmarked"
)

// Basic mailbox info.
type MailboxInfo struct {
	// The mailbox attributes.
	Attributes []string
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

	attrs, _ := fields[0].([]interface{})
	info.Attributes, _ = ParseStringList(attrs)

	info.Delimiter, _ = fields[1].(string)

	name, _ := fields[2].(string)
	info.Name, _ = utf7.Decoder.String(name)
	info.Name = CanonicalMailboxName(info.Name)

	return nil
}

// Format mailbox info to fields.
func (info *MailboxInfo) Format() []interface{} {
	name, _ := utf7.Encoder.String(info.Name)
	// Thunderbird doesn't understand delimiters if not quoted
	return []interface{}{FormatStringList(info.Attributes), Quoted(info.Delimiter), name}
}

// Mailbox status items.
const (
	MailboxFlags          = "FLAGS"
	MailboxPermanentFlags = "PERMANENTFLAGS"
	MailboxMessages       = "MESSAGES"
	MailboxRecent         = "RECENT"
	MailboxUnseen         = "UNSEEN"
	MailboxUidNext        = "UIDNEXT"
	MailboxUidValidity    = "UIDVALIDITY"
)

// A mailbox status.
type MailboxStatus struct {
	// The mailbox name.
	Name string
	// True if the mailbox is open in read-only mode.
	ReadOnly bool
	// The mailbox items that are currently filled in.
	Items []string

	// The mailbox flags.
	Flags []string
	// The mailbox permanent flags.
	PermanentFlags []string

	// The number of messages in this mailbox.
	Messages uint32
	// The number of messages not seen since the last time the mailbox was opened.
	Recent uint32
	// The number of unread messages.
	Unseen uint32
	// The next UID.
	UidNext uint32
	// Together with a UID, it is a unique identifier for a message.
	// Must be greater than or equal to 1.
	UidValidity uint32
}
