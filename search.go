package imap

import (
	"time"
)

// SearchReturnOption indicates what kind of results to return from a SEARCH
// command.
type SearchReturnOption string

const (
	SearchReturnMin   SearchReturnOption = "MIN"
	SearchReturnMax   SearchReturnOption = "MAX"
	SearchReturnAll   SearchReturnOption = "ALL"
	SearchReturnCount SearchReturnOption = "COUNT"
)

// SearchOptions contains options for the SEARCH command.
type SearchOptions struct {
	Return []SearchReturnOption // requires IMAP4rev2 or ESEARCH
}

// SearchCriteria is a criteria for the SEARCH command.
//
// When multiple fields are populated, the result is the intersection ("and"
// function) of all messages that match the fields.
type SearchCriteria struct {
	SeqNum SeqSet
	UID    SeqSet

	// Only the date is used, the time and timezone are ignored
	Since      time.Time
	Before     time.Time
	SentSince  time.Time
	SentBefore time.Time

	Header []SearchCriteriaHeaderField
	Body   []string
	Text   []string

	Flag    []Flag
	NotFlag []Flag

	Larger  int64
	Smaller int64

	Not []SearchCriteria
	Or  [][2]SearchCriteria

	ThreadId int64
}

type SearchCriteriaHeaderField struct {
	Key, Value string
}

// SearchData is the data returned by a SEARCH command.
type SearchData struct {
	All SeqSet

	// requires IMAP4rev2 or ESEARCH
	UID   bool
	Min   uint32
	Max   uint32
	Count uint32
}

// AllNums returns All as a slice of numbers.
func (data *SearchData) AllNums() []uint32 {
	// Note: a dynamic sequence set would be a server bug
	nums, _ := data.All.Nums()
	return nums
}
