package imap

const (
	// ResponseCodeNotificationOverflow is returned when the server cannot
	// handle the requested notifications (RFC 5465).
	ResponseCodeNotificationOverflow ResponseCode = "NOTIFICATIONOVERFLOW"
)

// NotifyEvent represents an event type for the NOTIFY command (RFC 5465).
type NotifyEvent string

const (
	// Message events
	NotifyEventFlagChange       NotifyEvent = "FlagChange"
	NotifyEventAnnotationChange NotifyEvent = "AnnotationChange"
	NotifyEventMessageNew       NotifyEvent = "MessageNew"
	NotifyEventMessageExpunge   NotifyEvent = "MessageExpunge"

	// Mailbox events
	NotifyEventMailboxName           NotifyEvent = "MailboxName"
	NotifyEventSubscriptionChange    NotifyEvent = "SubscriptionChange"
	NotifyEventMailboxMetadataChange NotifyEvent = "MailboxMetadataChange"
	NotifyEventServerMetadataChange  NotifyEvent = "ServerMetadataChange"
)

// NotifyMailboxSpec represents a mailbox specifier (rfc5465#section-6) for the NOTIFY command.
type NotifyMailboxSpec string

const (
	NotifyMailboxSpecSelected        NotifyMailboxSpec = "SELECTED"
	NotifyMailboxSpecSelectedDelayed NotifyMailboxSpec = "SELECTED-DELAYED"
	NotifyMailboxSpecPersonal        NotifyMailboxSpec = "PERSONAL"
	NotifyMailboxSpecInboxes         NotifyMailboxSpec = "INBOXES"
	NotifyMailboxSpecSubscribed      NotifyMailboxSpec = "SUBSCRIBED"
)

// NotifyOptions contains options for the NOTIFY command.
// XXX: how does this encode `NOTIFY NONE`?
type NotifyOptions struct {
	// STATUS indicates that a STATUS response should be sent for new mailboxes.
	// Only valid with Personal, Inboxes, or Subscribed mailbox specs.
	STATUS bool

	// Items represents the mailbox and events to monitor.
	Items []NotifyItem
}

// NotifyItem represents a mailbox or mailbox set and its events.
type NotifyItem struct {
	// MailboxSpec is a special mailbox specifier (Selected, Personal, etc.)
	// If empty, Mailboxes must be non-empty.
	MailboxSpec NotifyMailboxSpec

	// Mailboxes is a list of specific mailboxes to monitor.
	// Can include wildcards (*).
	Mailboxes []string

	// Subtree indicates that all mailboxes under the specified mailboxes
	// should be monitored (recursive).
	Subtree bool

	// Events is the list of events to monitor for these mailboxes.
	Events []NotifyEvent
}
