package imap

import (
	"fmt"
	"strings"
)

type RightSet string

type Right byte

const (
	RightLookup     = Right('l')
	RightRead       = Right('r')
	RightSeen       = Right('s')
	RightWrite      = Right('w')
	RightInsert     = Right('i')
	RightPost       = Right('p')
	RightCreate     = Right('c')
	RightDelete     = Right('d')
	RightAdminister = Right('a')

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
