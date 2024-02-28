package imap

import (
	"github.com/emersion/go-imap/v2/internal/imapnum"
)

// NumSet is a set of numbers identifying messages. NumSet is either a SeqSet
// or a UIDSet.
type NumSet interface {
	// String returns the IMAP representation of the message number set.
	String() string
	// Dynamic returns true if the set contains "*" or "n:*" ranges or if the
	// set represents the special SEARCHRES marker.
	Dynamic() bool
}

var (
	_ NumSet = SeqSet(nil)
	_ NumSet = UIDSet(nil)
)

// SeqSet is a set of message sequence numbers.
type SeqSet imapnum.Set[uint32]

// SeqSetNum returns a new SeqSet containing the specified sequence numbers.
func SeqSetNum(nums ...uint32) SeqSet {
	var s SeqSet
	s.AddNum(nums...)
	return s
}

func (s SeqSet) String() string {
	return imapnum.Set[uint32](s).String()
}

// Dynamic returns true if the set contains "*" or "n:*" values.
func (s SeqSet) Dynamic() bool {
	return imapnum.Set[uint32](s).Dynamic()
}

// Contains returns true if the non-zero sequence number is contained in the set.
func (s SeqSet) Contains(uid uint32) bool {
	return imapnum.Set[uint32](s).Contains(uid)
}

// Nums returns a slice of all sequence numbers contained in the set.
func (s SeqSet) Nums() ([]uint32, bool) {
	return imapnum.Set[uint32](s).Nums()
}

// AddNum inserts new sequence numbers into the set. The value 0 represents "*".
func (s *SeqSet) AddNum(uids ...uint32) {
	(*imapnum.Set[uint32])(s).AddNum(uids...)
}

// AddRange inserts a new range into the set.
func (s *SeqSet) AddRange(start, stop uint32) {
	(*imapnum.Set[uint32])(s).AddRange(start, stop)
}

// AddSet inserts all sequence numbers from other into s.
func (s *SeqSet) AddSet(other SeqSet) {
	(*imapnum.Set[uint32])(s).AddSet(imapnum.Set[uint32](other))
}

// SeqRange is a range of message sequence numbers.
type SeqRange = imapnum.Range[uint32]

// UIDSet is a set of message UIDs.
type UIDSet imapnum.Set[UID]

// UIDSetNum returns a new UIDSet containing the specified UIDs.
func UIDSetNum(uids ...UID) UIDSet {
	var s UIDSet
	s.AddNum(uids...)
	return s
}

func (s UIDSet) String() string {
	if IsSearchRes(s) {
		return "$"
	}
	return imapnum.Set[UID](s).String()
}

// Dynamic returns true if the set contains "*" or "n:*" values.
func (s UIDSet) Dynamic() bool {
	return imapnum.Set[UID](s).Dynamic() || IsSearchRes(s)
}

// Contains returns true if the non-zero UID is contained in the set.
func (s UIDSet) Contains(uid UID) bool {
	return imapnum.Set[UID](s).Contains(uid)
}

// Nums returns a slice of all UIDs contained in the set.
func (s UIDSet) Nums() ([]UID, bool) {
	return imapnum.Set[UID](s).Nums()
}

// AddNum inserts new UIDs into the set. The value 0 represents "*".
func (s *UIDSet) AddNum(uids ...UID) {
	(*imapnum.Set[UID])(s).AddNum(uids...)
}

// AddRange inserts a new range into the set.
func (s *UIDSet) AddRange(start, stop UID) {
	(*imapnum.Set[UID])(s).AddRange(start, stop)
}

// AddSet inserts all UIDs from other into s.
func (s *UIDSet) AddSet(other UIDSet) {
	(*imapnum.Set[UID])(s).AddSet(imapnum.Set[UID](other))
}

// UIDRange is a range of message UIDs.
type UIDRange = imapnum.Range[UID]
