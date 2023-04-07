package imap

import (
	"time"
)

// SearchOptions contains options for the SEARCH command.
type SearchOptions struct {
	// Requires IMAP4rev2 or ESEARCH
	ReturnMin   bool
	ReturnMax   bool
	ReturnAll   bool
	ReturnCount bool
}

// SearchCriteria is a criteria for the SEARCH command.
//
// When multiple fields are populated, the result is the intersection ("and"
// function) of all messages that match the fields.
type SearchCriteria struct {
	SeqNum []SeqSet
	UID    []SeqSet

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

	Not *SearchCriteria
	Or  [][2]SearchCriteria
}

// And intersects two search criteria.
func (criteria *SearchCriteria) And(other *SearchCriteria) {
	criteria.SeqNum = append(criteria.SeqNum, other.SeqNum...)
	criteria.UID = append(criteria.UID, other.UID...)

	criteria.Since = intersectSince(criteria.Since, other.Since)
	criteria.Before = intersectBefore(criteria.Before, other.Before)
	criteria.SentSince = intersectSince(criteria.SentSince, other.SentSince)
	criteria.SentBefore = intersectBefore(criteria.SentBefore, other.SentBefore)

	criteria.Header = append(criteria.Header, other.Header...)
	criteria.Body = append(criteria.Body, other.Body...)
	criteria.Text = append(criteria.Text, other.Text...)

	criteria.Flag = append(criteria.Flag, other.Flag...)
	criteria.NotFlag = append(criteria.NotFlag, other.NotFlag...)

	if criteria.Larger == 0 || other.Larger > criteria.Larger {
		criteria.Larger = other.Larger
	}
	if criteria.Smaller == 0 || other.Smaller < criteria.Smaller {
		criteria.Smaller = other.Smaller
	}

	if criteria.Not != nil && other.Not != nil {
		criteria.Not.And(other.Not)
	} else if other.Not != nil {
		criteria.Not = other.Not
	}
	criteria.Or = append(criteria.Or, other.Or...)
}

func intersectSince(t1, t2 time.Time) time.Time {
	switch {
	case t1.IsZero():
		return t2
	case t2.IsZero():
		return t1
	case t1.After(t2):
		return t1
	default:
		return t2
	}
}

func intersectBefore(t1, t2 time.Time) time.Time {
	switch {
	case t1.IsZero():
		return t2
	case t2.IsZero():
		return t1
	case t1.Before(t2):
		return t1
	default:
		return t2
	}
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
