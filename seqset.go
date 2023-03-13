package imap

import (
	"fmt"
	"strconv"
	"strings"
)

// errBadSeqSet is used to report problems with the format of a sequence set
// value.
type errBadSeqSet string

func (err errBadSeqSet) Error() string {
	return fmt.Sprintf("imap: bad sequence set value %q", string(err))
}

// Seq represents a single seq-number or seq-range value (RFC 3501 ABNF). Values
// may be static (e.g. "1", "2:4") or dynamic (e.g. "*", "1:*"). A seq-number is
// represented by setting Start = Stop. Zero is used to represent "*", which is
// safe because seq-number uses nz-number rule. The order of values is always
// Start <= Stop, except when representing "n:*", where Start = n and Stop = 0.
type Seq struct {
	Start, Stop uint32
}

// parseSeqNum parses a single seq-number value (non-zero uint32 or "*").
func parseSeqNum(v string) (uint32, error) {
	if n, err := strconv.ParseUint(v, 10, 32); err == nil && v[0] != '0' {
		return uint32(n), nil
	} else if v == "*" {
		return 0, nil
	}
	return 0, errBadSeqSet(v)
}

// parseSeq creates a new seq instance by parsing strings in the format "n" or
// "n:m", where n and/or m may be "*". An error is returned for invalid values.
func parseSeq(v string) (Seq, error) {
	var (
		s   Seq
		err error
	)
	if sep := strings.IndexRune(v, ':'); sep < 0 {
		s.Start, err = parseSeqNum(v)
		s.Stop = s.Start
		return s, err
	} else if s.Start, err = parseSeqNum(v[:sep]); err == nil {
		if s.Stop, err = parseSeqNum(v[sep+1:]); err == nil {
			if (s.Stop < s.Start && s.Stop != 0) || s.Start == 0 {
				s.Start, s.Stop = s.Stop, s.Start
			}
			return s, nil
		}
	}
	return s, errBadSeqSet(v)
}

// Contains returns true if the seq-number q is contained in sequence value s.
// The dynamic value "*" contains only other "*" values, the dynamic range "n:*"
// contains "*" and all numbers >= n.
func (s Seq) Contains(q uint32) bool {
	if q == 0 {
		return s.Stop == 0 // "*" is contained only in "*" and "n:*"
	}
	return s.Start != 0 && s.Start <= q && (q <= s.Stop || s.Stop == 0)
}

// Less returns true if s precedes and does not contain seq-number q.
func (s Seq) Less(q uint32) bool {
	return (s.Stop < q || q == 0) && s.Stop != 0
}

// Merge combines sequence values s and t into a single union if the two
// intersect or one is a superset of the other. The order of s and t does not
// matter. If the values cannot be merged, s is returned unmodified and ok is
// set to false.
func (s Seq) Merge(t Seq) (union Seq, ok bool) {
	union = s
	if s == t {
		return s, true
	}
	if s.Start != 0 && t.Start != 0 {
		// s and t are any combination of "n", "n:m", or "n:*"
		if s.Start > t.Start {
			s, t = t, s
		}
		// s starts at or before t, check where it ends
		if (s.Stop >= t.Stop && t.Stop != 0) || s.Stop == 0 {
			return s, true // s is a superset of t
		}
		// s is "n" or "n:m", if m == ^uint32(0) then t is "n:*"
		if s.Stop+1 >= t.Start || s.Stop == ^uint32(0) {
			return Seq{s.Start, t.Stop}, true // s intersects or touches t
		}
		return union, false
	}
	// exactly one of s and t is "*"
	if s.Start == 0 {
		if t.Stop == 0 {
			return t, true // s is "*", t is "n:*"
		}
	} else if s.Stop == 0 {
		return s, true // s is "n:*", t is "*"
	}
	return union, false
}

// String returns sequence value s as a seq-number or seq-range string.
func (s Seq) String() string {
	if s.Start == s.Stop {
		if s.Start == 0 {
			return "*"
		}
		return strconv.FormatUint(uint64(s.Start), 10)
	}
	b := strconv.AppendUint(make([]byte, 0, 24), uint64(s.Start), 10)
	if s.Stop == 0 {
		return string(append(b, ':', '*'))
	}
	return string(strconv.AppendUint(append(b, ':'), uint64(s.Stop), 10))
}

func (s Seq) append(nums []uint32) (out []uint32, ok bool) {
	if s.Start == 0 || s.Stop == 0 {
		return nil, false
	}
	for n := s.Start; n <= s.Stop; n++ {
		nums = append(nums, n)
	}
	return nums, true
}

// SeqSet is used to represent a set of message sequence numbers or UIDs (see
// sequence-set ABNF rule). The zero value is an empty set.
type SeqSet []Seq

// ParseSeqSet returns a new SeqSet after parsing the set string.
func ParseSeqSet(set string) (SeqSet, error) {
	var s SeqSet
	for _, sv := range strings.Split(set, ",") {
		v, err := parseSeq(sv)
		if err != nil {
			return s, err
		}
		s.insert(v)
	}
	return s, nil
}

// SeqSetNum returns a new SeqSet containing the sequence numbers.
func SeqSetNum(q ...uint32) SeqSet {
	var s SeqSet
	s.AddNum(q...)
	return s
}

// SeqSetRange returns a new SeqSet containing the sequence range.
func SeqSetRange(start, stop uint32) SeqSet {
	var s SeqSet
	s.AddRange(start, stop)
	return s
}

// AddNum inserts new sequence numbers into the set. The value 0 represents "*".
func (s *SeqSet) AddNum(q ...uint32) {
	for _, v := range q {
		s.insert(Seq{v, v})
	}
}

// AddRange inserts a new sequence range into the set.
func (s *SeqSet) AddRange(start, stop uint32) {
	if (stop < start && stop != 0) || start == 0 {
		s.insert(Seq{stop, start})
	} else {
		s.insert(Seq{start, stop})
	}
}

// AddSet inserts all values from t into s.
func (s *SeqSet) AddSet(t SeqSet) {
	for _, v := range t {
		s.insert(v)
	}
}

// Dynamic returns true if the set contains "*" or "n:*" values.
func (s SeqSet) Dynamic() bool {
	return len(s) > 0 && s[len(s)-1].Stop == 0
}

// Contains returns true if the non-zero sequence number or UID q is contained
// in the set. The dynamic range "n:*" contains all q >= n. It is the caller's
// responsibility to handle the special case where q is the maximum UID in the
// mailbox and q < n (i.e. the set cannot match UIDs against "*:n" or "*" since
// it doesn't know what the maximum value is).
func (s SeqSet) Contains(q uint32) bool {
	if _, ok := s.search(q); ok {
		return q != 0
	}
	return false
}

// Nums returns a slice of all numbers contained in the sequence set.
func (s SeqSet) Nums() (nums []uint32, ok bool) {
	for _, v := range s {
		nums, ok = v.append(nums)
		if !ok {
			return nil, false
		}
	}
	return nums, true
}

// String returns a sorted representation of all contained sequence values.
func (s SeqSet) String() string {
	if len(s) == 0 {
		return ""
	}
	b := make([]byte, 0, 64)
	for _, v := range s {
		b = append(b, ',')
		if v.Start == 0 {
			b = append(b, '*')
			continue
		}
		b = strconv.AppendUint(b, uint64(v.Start), 10)
		if v.Start != v.Stop {
			if v.Stop == 0 {
				b = append(b, ':', '*')
				continue
			}
			b = strconv.AppendUint(append(b, ':'), uint64(v.Stop), 10)
		}
	}
	return string(b[1:])
}

// insert adds sequence value v to the set.
func (ptr *SeqSet) insert(v Seq) {
	s := *ptr
	defer func() {
		*ptr = s
	}()

	i, _ := s.search(v.Start)
	merged := false
	if i > 0 {
		// try merging with the preceding entry (e.g. "1,4".insert(2), i == 1)
		s[i-1], merged = s[i-1].Merge(v)
	}
	if i == len(s) {
		// v was either merged with the last entry or needs to be appended
		if !merged {
			s.insertAt(i, v)
		}
		return
	} else if merged {
		i--
	} else if s[i], merged = s[i].Merge(v); !merged {
		s.insertAt(i, v) // insert in the middle (e.g. "1,5".insert(3), i == 1)
		return
	}
	// v was merged with s[i], continue trying to merge until the end
	for j := i + 1; j < len(s); j++ {
		if s[i], merged = s[i].Merge(s[j]); !merged {
			if j > i+1 {
				// cut out all entries between i and j that were merged
				s = append(s[:i+1], s[j:]...)
			}
			return
		}
	}
	// everything after s[i] was merged
	s = s[:i+1]
}

// insertAt inserts a new sequence value v at index i, resizing s.Set as needed.
func (ptr *SeqSet) insertAt(i int, v Seq) {
	s := *ptr
	defer func() {
		*ptr = s
	}()

	if n := len(s); i == n {
		// insert at the end
		s = append(s, v)
		return
	} else if n < cap(s) {
		// enough space, shift everything at and after i to the right
		s = s[:n+1]
		copy(s[i+1:], s[i:])
	} else {
		// allocate new slice and copy everything, n is at least 1
		set := make([]Seq, n+1, n*2)
		copy(set, s[:i])
		copy(set[i+1:], s[i:])
		s = set
	}
	s[i] = v
}

// search attempts to find the index of the sequence set value that contains q.
// If no values contain q, the returned index is the position where q should be
// inserted and ok is set to false.
func (s SeqSet) search(q uint32) (i int, ok bool) {
	min, max := 0, len(s)-1
	for min < max {
		if mid := (min + max) >> 1; s[mid].Less(q) {
			min = mid + 1
		} else {
			max = mid
		}
	}
	if max < 0 || s[min].Less(q) {
		return len(s), false // q is the new largest value
	}
	return min, s[min].Contains(q)
}
