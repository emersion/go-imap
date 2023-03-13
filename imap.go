// Package IMAP implements IMAP4rev2.
//
// IMAP4rev2 is defined in RFC 9051.
package imap

// MailboxAttr is a mailbox attribute.
//
// Mailbox attributes are defined in RFC 9051 section 7.3.1.
type MailboxAttr string

const (
	// Base attributes
	MailboxAttrNonExistent   MailboxAttr = "\\NonExistent"
	MailboxAttrNoInferiors   MailboxAttr = "\\Noinferiors"
	MailboxAttrNoSelect      MailboxAttr = "\\Noselect"
	MailboxAttrHasChildren   MailboxAttr = "\\HasChildren"
	MailboxAttrHasNoChildren MailboxAttr = "\\HasNoChildren"
	MailboxAttrMarked        MailboxAttr = "\\Marked"
	MailboxAttrUnmarked      MailboxAttr = "\\Unmarked"
	MailboxAttrSubscribed    MailboxAttr = "\\Subscribed"
	MailboxAttrRemote        MailboxAttr = "\\Remote"

	// Role (aka. "special-use") attributes
	MailboxAttrAll     MailboxAttr = "\\All"
	MailboxAttrArchive MailboxAttr = "\\Archive"
	MailboxAttrDrafts  MailboxAttr = "\\Drafts"
	MailboxAttrFlagged MailboxAttr = "\\Flagged"
	MailboxAttrJunk    MailboxAttr = "\\Junk"
	MailboxAttrSent    MailboxAttr = "\\Sent"
	MailboxAttrTrash   MailboxAttr = "\\Trash"
)

// Flag is a message flag.
//
// Message flags are defined in RFC 9051 section 2.3.2.
type Flag string

const (
	// System flags
	FlagSeen     Flag = "\\Seen"
	FlagAnswered Flag = "\\Answered"
	FlagFlagged  Flag = "\\Flagged"
	FlagDeleted  Flag = "\\Deleted"
	FlagDraft    Flag = "\\Draft"

	// Widely used flags
	FlagForwarded Flag = "$Forwarded"
	FlagMDNSent   Flag = "$MDNSent" // Message Disposition Notification sent
	FlagJunk      Flag = "$Junk"
	FlagNotJunk   Flag = "$NotJunk"
	FlagPhishing  Flag = "$Phishing"
	FlagImportant Flag = "$Important" // RFC 8457

	// Permanent flags
	FlagWildcard Flag = "\\*"
)
