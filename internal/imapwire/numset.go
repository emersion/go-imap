package imapwire

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emersion/go-imap/v2"
)

// errBadNumSet is used to report problems with the format of a number set
// value.
type errBadNumSet string

func (err errBadNumSet) Error() string {
	return fmt.Sprintf("imap: bad number set value %q", string(err))
}

// parseNum parses a single seq-number value (non-zero uint32 or "*").
func parseNum(v string) (uint32, error) {
	if n, err := strconv.ParseUint(v, 10, 32); err == nil && v[0] != '0' {
		return uint32(n), nil
	} else if v == "*" {
		return 0, nil
	}
	return 0, errBadNumSet(v)
}

// parseNumRange creates a new seq instance by parsing strings in the format
// "n" or "n:m", where n and/or m may be "*". An error is returned for invalid
// values.
func parseNumRange(v string) (imap.NumRange, error) {
	var (
		s   imap.NumRange
		err error
	)
	if sep := strings.IndexRune(v, ':'); sep < 0 {
		s.Start, err = parseNum(v)
		s.Stop = s.Start
		return s, err
	} else if s.Start, err = parseNum(v[:sep]); err == nil {
		if s.Stop, err = parseNum(v[sep+1:]); err == nil {
			if (s.Stop < s.Start && s.Stop != 0) || s.Start == 0 {
				s.Start, s.Stop = s.Stop, s.Start
			}
			return s, nil
		}
	}
	return s, errBadNumSet(v)
}

// ParseNumSet returns a new NumSet after parsing the set string.
func ParseNumSet(set string) (imap.NumSet, error) {
	var s imap.NumSet
	for _, sv := range strings.Split(set, ",") {
		v, err := parseNumRange(sv)
		if err != nil {
			return s, err
		}
		s.AddRange(v.Start, v.Stop)
	}
	return s, nil
}
