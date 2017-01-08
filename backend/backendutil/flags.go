package backendutil

import (
	"github.com/emersion/go-imap"
)

// UpdateFlags executes a flag operation on the flag set current.
func UpdateFlags(current []string, op imap.FlagsOp, flags []string) []string {
	switch op {
	case imap.SetFlags:
		// TODO: keep \Recent if it is present
		return flags
	case imap.AddFlags:
		// Check for duplicates
		for _, flag := range current {
			for i, addFlag := range flags {
				if addFlag == flag {
					flags = append(flags[:i], flags[i+1:]...)
					break
				}
			}
		}
		return append(current, flags...)
	case imap.RemoveFlags:
		// Iterate through flags from the last one to the first one, to be able to
		// delete some of them.
		for i := len(current) - 1; i >= 0; i-- {
			flag := current[i]

			for _, removeFlag := range flags {
				if removeFlag == flag {
					current = append(current[:i], current[i+1:]...)
					break
				}
			}
		}
		return current
	}
	return current
}
