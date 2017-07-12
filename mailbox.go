package imap

import (
	"errors"
	"strings"
	"sync"

	"github.com/emersion/go-imap/utf7"
)

// The primary mailbox, as defined in RFC 3501 section 5.1.
const InboxName = "INBOX"

// Returns the canonical form of a mailbox name. Mailbox names can be
// case-sensitive or case-insensitive depending on the backend implementation.
// The special INBOX mailbox is case-insensitive.
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

	var err error
	if info.Attributes, err = ParseStringList(fields[0]); err != nil {
		return err
	}

	var ok bool
	if info.Delimiter, ok = fields[1].(string); !ok {
		return errors.New("Mailbox delimiter must be a string")
	}

	if name, err := ParseString(fields[2]); err != nil {
		return err
	} else if name, err := utf7.Encoding.NewDecoder().String(name); err != nil {
		return err
	} else {
		info.Name = CanonicalMailboxName(name)
	}

	return nil
}

// Format mailbox info to fields.
func (info *MailboxInfo) Format() []interface{} {
	name, _ := utf7.Encoding.NewEncoder().String(info.Name)
	// Thunderbird doesn't understand delimiters if not quoted
	return []interface{}{FormatStringList(info.Attributes), Quoted(info.Delimiter), name}
}

// TODO: optimize this
func (info *MailboxInfo) match(name, pattern string) bool {
	i := strings.IndexAny(pattern, "*%")
	if i == -1 {
		// No more wildcards
		return name == pattern
	}

	// Get parts before and after wildcard
	chunk, wildcard, rest := pattern[0:i], pattern[i], pattern[i+1:]

	// Check that name begins with chunk
	if len(chunk) > 0 && !strings.HasPrefix(name, chunk) {
		return false
	}
	name = strings.TrimPrefix(name, chunk)

	// Expand wildcard
	var j int
	for j = 0; j < len(name); j++ {
		if wildcard == '%' && string(name[j]) == info.Delimiter {
			break // Stop on delimiter if wildcard is %
		}
		// Try to match the rest from here
		if info.match(name[j:], rest) {
			return true
		}
	}

	return info.match(name[j:], rest)
}

// Match checks if a reference and a pattern matches this mailbox name, as
// defined in RFC 3501 section 6.3.8.
func (info *MailboxInfo) Match(reference, pattern string) bool {
	name := info.Name

	if strings.HasPrefix(pattern, info.Delimiter) {
		reference = ""
		pattern = strings.TrimPrefix(pattern, info.Delimiter)
	}
	if reference != "" {
		if !strings.HasSuffix(reference, info.Delimiter) {
			reference += info.Delimiter
		}
		if !strings.HasPrefix(name, reference) {
			return false
		}
		name = strings.TrimPrefix(name, reference)
	}

	return info.match(name, pattern)
}

// Mailbox status items.
const (
	MailboxFlags          = "FLAGS"
	MailboxPermanentFlags = "PERMANENTFLAGS"

	// Defined in RFC 3501 section 6.3.10.
	MailboxMessages    = "MESSAGES"
	MailboxRecent      = "RECENT"
	MailboxUnseen      = "UNSEEN"
	MailboxUidNext     = "UIDNEXT"
	MailboxUidValidity = "UIDVALIDITY"
)

// A mailbox status.
type MailboxStatus struct {
	// The mailbox name.
	Name string
	// True if the mailbox is open in read-only mode.
	ReadOnly bool
	// The mailbox items that are currently filled in. This map's values
	// should not be used directly, they must only be used by libraries
	// implementing extensions of the IMAP protocol.
	Items map[string]interface{}

	// The Items map may be accessed in different goroutines. Protect
	// concurrent writes.
	ItemsLocker sync.Mutex

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

// Create a new mailbox status that will contain the specified items.
func NewMailboxStatus(name string, items []string) *MailboxStatus {
	status := &MailboxStatus{
		Name:  name,
		Items: make(map[string]interface{}),
	}

	for _, k := range items {
		status.Items[k] = nil
	}

	return status
}

func (status *MailboxStatus) Parse(fields []interface{}) error {
	status.Items = make(map[string]interface{})

	var k string
	for i, f := range fields {
		if i%2 == 0 {
			var ok bool
			if k, ok = f.(string); !ok {
				return errors.New("Key is not a string")
			}
			k = strings.ToUpper(k)
		} else {
			status.Items[k] = nil

			var err error
			switch k {
			case MailboxMessages:
				status.Messages, err = ParseNumber(f)
			case MailboxRecent:
				status.Recent, err = ParseNumber(f)
			case MailboxUnseen:
				status.Unseen, err = ParseNumber(f)
			case MailboxUidNext:
				status.UidNext, err = ParseNumber(f)
			case MailboxUidValidity:
				status.UidValidity, err = ParseNumber(f)
			default:
				status.Items[k] = f
			}

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (status *MailboxStatus) Format() []interface{} {
	var fields []interface{}
	for k, v := range status.Items {
		switch k {
		case MailboxMessages:
			v = status.Messages
		case MailboxRecent:
			v = status.Recent
		case MailboxUnseen:
			v = status.Unseen
		case MailboxUidNext:
			v = status.UidNext
		case MailboxUidValidity:
			v = status.UidValidity
		}

		fields = append(fields, k, v)
	}
	return fields
}
