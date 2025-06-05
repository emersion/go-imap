package imap

// SortKey represents a sort key for the SORT command.
type SortKey string

// Predefined sort keys from RFC 5256.
const (
	SortKeyArrival SortKey = "ARRIVAL"
	SortKeyCc      SortKey = "CC"
	SortKeyDate    SortKey = "DATE"
	SortKeyFrom    SortKey = "FROM"
	SortKeySize    SortKey = "SIZE"
	SortKeySubject SortKey = "SUBJECT"
	SortKeyTo      SortKey = "TO"
)

// SortCriterion represents a single sort criterion for the SORT command.
type SortCriterion struct {
	Key     SortKey
	Reverse bool
}

// SortOptions contains options for the SORT command.
type SortOptions struct {
	// DisplayLocale is the locale for SORT=DISPLAY extension (RFC 5957)
	// This field is only used if the server supports the SORT=DISPLAY capability.
	DisplayLocale string
}
