package imap

import (
	"fmt"
	"strings"
)

type RightSet string

type Right byte

const (
	RightLookup     = Right('l') // mailbox is visible to LIST/LSUB commands
	RightRead       = Right('r') // SELECT the mailbox, perform CHECK, FETCH, PARTIAL, SEARCH, COPY from mailbox
	RightSeen       = Right('s') // keep seen/unseen information across sessions (STORE SEEN flag)
	RightWrite      = Right('w') // STORE flags other than SEEN and DELETED
	RightInsert     = Right('i') // perform APPEND, COPY into mailbox
	RightPost       = Right('p') // send mail to submission address for mailbox, not enforced by IMAP4 itself
	RightCreate     = Right('c') // CREATE new sub-mailboxes in any implementation-defined hierarchy
	RightDelete     = Right('d') // STORE DELETED flag, perform EXPUNGE
	RightAdminister = Right('a') // perform SETACL

	AllRights = RightSet("lrswipcda")
)

type RightsIdentifier string

const RightsIdentifierAnyone = RightsIdentifier("anyone")

// NewRightSet converts rights string into RightSet with validation
func NewRightSet(rights string) (RightSet, error) {
	for _, r := range rights {
		if !strings.ContainsRune(string(AllRights), r) {
			return "", fmt.Errorf("unsupported right: '%v'", string(r))
		}
	}

	return RightSet(rights), nil
}

func (r RightSet) Add(rights RightSet) RightSet {
	for _, right := range rights {
		if !strings.ContainsRune(string(r), right) {
			r += RightSet(right)
		}
	}

	return r
}

func (r RightSet) Remove(rights RightSet) RightSet {
	var newRights RightSet

	for _, right := range r {
		if !strings.ContainsRune(string(rights), right) {
			newRights += RightSet(right)
		}
	}

	return newRights
}
