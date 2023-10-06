package imap

import (
	"reflect"
	"time"
)

// SearchOptions contains options for the SEARCH command.
type SearchOptions struct {
	// Requires IMAP4rev2 or ESEARCH
	ReturnMin   bool
	ReturnMax   bool
	ReturnAll   bool
	ReturnCount bool
	// Requires IMAP4rev2 or SEARCHRES
	ReturnSave bool
}

// SearchCriteria is a criteria for the SEARCH command.
//
// When multiple fields are populated, the result is the intersection ("and"
// function) of all messages that match the fields.
//
// And, Not and Or can be used to combine multiple criteria together. For
// instance, the following criteria matches messages not containing "hello":
//
//	SearchCriteria{Not: []SearchCriteria{{
//		Body: []string{"hello"},
//	}}}
//
// The following criteria matches messages containing either "hello" or
// "world":
//
//	SearchCriteria{Or: [][2]SearchCriteria{{
//		{Body: []string{"hello"}},
//		{Body: []string{"world"}},
//	}}}
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

	Not []SearchCriteria
	Or  [][2]SearchCriteria

	ModSeq *SearchCriteriaModSeq // requires CONDSTORE
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

	criteria.Not = append(criteria.Not, other.Not...)
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

type SearchCriteriaModSeq struct {
	ModSeq       uint64
	MetadataName string
	MetadataType SearchCriteriaMetadataType
}

type SearchCriteriaMetadataType string

const (
	SearchCriteriaMetadataAll     SearchCriteriaMetadataType = "all"
	SearchCriteriaMetadataPrivate SearchCriteriaMetadataType = "priv"
	SearchCriteriaMetadataShared  SearchCriteriaMetadataType = "shared"
)

// SearchData is the data returned by a SEARCH command.
type SearchData struct {
	All SeqSet

	// requires IMAP4rev2 or ESEARCH
	UID   bool
	Min   uint32
	Max   uint32
	Count uint32

	// requires CONDSTORE
	ModSeq uint64
}

// AllNums returns All as a slice of numbers.
func (data *SearchData) AllNums() []uint32 {
	// Note: a dynamic sequence set would be a server bug
	nums, _ := data.All.Nums()
	return nums
}

// searchRes is a special empty SeqSet which can be used as a marker. It has
// a non-zero cap so that its data pointer is non-nil and can be compared.
var (
	searchRes     = make(SeqSet, 0, 1)
	searchResAddr = reflect.ValueOf(searchRes).Pointer()
)

// SearchRes returns a special marker which can be used instead of a SeqSet to
// reference the last SEARCH result. On the wire, it's encoded as '$'.
//
// It requires IMAP4rev2 or the SEARCHRES extension.
func SearchRes() SeqSet {
	return searchRes
}

// IsSearchRes checks whether a sequence set is a reference to the last SEARCH
// result. See SearchRes.
func IsSearchRes(seqSet SeqSet) bool {
	return reflect.ValueOf(seqSet).Pointer() == searchResAddr
}
