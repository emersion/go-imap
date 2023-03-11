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
