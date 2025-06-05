package imapmemserver

import (
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

func (s *UserSession) Sort(kind imapserver.NumKind, sortCriteria []imap.SortCriterion, criteria *imap.SearchCriteria, options *imap.SearchOptions, sortOptions *imap.SortOptions) (*imap.SearchData, error) {
	// Use the search function to find matching messages
	searchData, err := s.Search(kind, criteria, options)
	if err != nil {
		return nil, err
	}

	// No need to sort if we don't have messages
	if isNumSetEmpty(searchData.All) {
		return searchData, nil
	}

	// Get the messages matching the search criteria
	var msgs []*message
	var nums []uint32
	var uids []imap.UID

	switch numSet := searchData.All.(type) {
	case imap.SeqSet:
		nums, _ = numSet.Nums()
		for _, num := range nums {
			msgs = append(msgs, s.mailbox.l[num-1])
		}
	case imap.UIDSet:
		uids, _ = numSet.Nums()
		for _, uid := range uids {
			for _, msg := range s.mailbox.l {
				if msg.uid == uid {
					msgs = append(msgs, msg)
					break
				}
			}
		}
	}

	// Sort the messages based on the provided criteria
	sort.SliceStable(msgs, func(i, j int) bool {
		return compareMsgs(msgs[i], msgs[j], sortCriteria, sortOptions)
	})

	// Update the sort result
	if kind == imapserver.NumKindSeq {
		var newSeqSet imap.SeqSet
		for i, msg := range msgs {
			seqNum := uint32(i) + 1
			for j, m := range s.mailbox.l {
				if m == msg {
					seqNum = uint32(j) + 1
					break
				}
			}
			newSeqSet = append(newSeqSet, imap.SeqRange{
				Start: seqNum,
				Stop:  seqNum,
			})
		}
		searchData.All = newSeqSet
	} else {
		var newUIDSet imap.UIDSet
		for _, msg := range msgs {
			newUIDSet = append(newUIDSet, imap.UIDRange{
				Start: msg.uid,
				Stop:  msg.uid,
			})
		}
		searchData.All = newUIDSet
	}

	// Update Min, Max, Count if requested
	// Recalculate since the order might have changed
	switch numSet := searchData.All.(type) {
	case imap.SeqSet:
		nums, _ = numSet.Nums()
		if len(nums) > 0 {
			searchData.Min = nums[0]
			searchData.Max = nums[len(nums)-1]
		}
	case imap.UIDSet:
		uids, _ = numSet.Nums()
		if len(uids) > 0 {
			searchData.Min = uint32(uids[0])
			searchData.Max = uint32(uids[len(uids)-1])
		}
	}
	searchData.Count = uint32(len(msgs))

	return searchData, nil
}

// Helper function to compare messages based on sort criteria
func compareMsgs(msg1, msg2 *message, criteria []imap.SortCriterion, sortOptions *imap.SortOptions) bool {
	for _, criterion := range criteria {
		var result int

		// Use locale-aware comparison if DISPLAY option is specified
		if sortOptions != nil && sortOptions.DisplayLocale != "" {
			result = compareByDisplayCriterion(msg1, msg2, criterion.Key, sortOptions.DisplayLocale)
		} else {
			result = compareByCriterion(msg1, msg2, criterion.Key)
		}

		// Apply reverse if needed
		if criterion.Reverse {
			result = -result
		}

		if result < 0 {
			return true
		} else if result > 0 {
			return false
		}
		// If equal, continue to the next criterion
	}

	// If all criteria resulted in equality, keep original order
	return false
}

// Helper function to compare messages with locale awareness (SORT=DISPLAY)
func compareByDisplayCriterion(msg1, msg2 *message, key imap.SortKey, locale string) int {
	// In a real implementation, this would use the locale to perform
	// collation-based string comparison using a library like ICU or golang.org/x/text

	// For this implementation, we'll just do a simple case-insensitive comparison
	// as a placeholder for proper locale-based sorting
	switch key {
	case imap.SortKeyFrom:
		from1 := getHeader(msg1, "From")
		from2 := getHeader(msg2, "From")
		cmp := strings.Compare(strings.ToLower(from1), strings.ToLower(from2))
		return cmp
	case imap.SortKeyTo:
		to1 := getHeader(msg1, "To")
		to2 := getHeader(msg2, "To")
		cmp := strings.Compare(strings.ToLower(to1), strings.ToLower(to2))
		return cmp
	case imap.SortKeyCc:
		cc1 := getHeader(msg1, "Cc")
		cc2 := getHeader(msg2, "Cc")
		cmp := strings.Compare(strings.ToLower(cc1), strings.ToLower(cc2))
		return cmp
	case imap.SortKeySubject:
		subject1 := getHeader(msg1, "Subject")
		subject2 := getHeader(msg2, "Subject")
		cmp := strings.Compare(strings.ToLower(subject1), strings.ToLower(subject2))
		return cmp
	default:
		// For non-string comparisons, fallback to the regular comparison
		return compareByCriterion(msg1, msg2, key)
	}
}

// Compare two messages based on a specific sort key
func compareByCriterion(msg1, msg2 *message, key imap.SortKey) int {
	switch key {
	case imap.SortKeyArrival:
		// Compare by internal date
		if msg1.t.Before(msg2.t) {
			return -1
		} else if msg1.t.After(msg2.t) {
			return 1
		}

	case imap.SortKeyDate:
		// Compare by date header
		date1 := getDateHeader(msg1)
		date2 := getDateHeader(msg2)
		if date1.Before(date2) {
			return -1
		} else if date1.After(date2) {
			return 1
		}

	case imap.SortKeyFrom:
		// Compare by FROM header
		from1 := getHeader(msg1, "From")
		from2 := getHeader(msg2, "From")
		return strings.Compare(from1, from2)

	case imap.SortKeyTo:
		// Compare by TO header
		to1 := getHeader(msg1, "To")
		to2 := getHeader(msg2, "To")
		return strings.Compare(to1, to2)

	case imap.SortKeyCc:
		// Compare by CC header
		cc1 := getHeader(msg1, "Cc")
		cc2 := getHeader(msg2, "Cc")
		return strings.Compare(cc1, cc2)

	case imap.SortKeySubject:
		// Compare by SUBJECT header
		subject1 := getHeader(msg1, "Subject")
		subject2 := getHeader(msg2, "Subject")
		return strings.Compare(subject1, subject2)

	case imap.SortKeySize:
		// Compare by message size
		if len(msg1.buf) < len(msg2.buf) {
			return -1
		} else if len(msg1.buf) > len(msg2.buf) {
			return 1
		}
	}

	// Default: equal
	return 0
}

// Helper to get header value
func getHeader(msg *message, name string) string {
	env := msg.envelope()
	if env == nil {
		return ""
	}

	switch strings.ToLower(name) {
	case "subject":
		if env.Subject != "" {
			return env.Subject
		}
	case "from":
		if len(env.From) > 0 {
			return formatAddress(env.From[0])
		}
	case "to":
		if len(env.To) > 0 {
			return formatAddress(env.To[0])
		}
	case "cc":
		if len(env.Cc) > 0 {
			return formatAddress(env.Cc[0])
		}
	}

	// Try to get from headers directly
	r := msg.reader()
	if r != nil {
		fields := r.Header.Fields()
		for fields.Next() {
			if strings.EqualFold(fields.Key(), name) {
				value, _ := fields.Text()
				return value
			}
		}
	}

	return ""
}

// Helper to format address
func formatAddress(addr imap.Address) string {
	var sb strings.Builder
	if addr.Name != "" {
		sb.WriteString(addr.Name)
	}

	sb.WriteString(" <")
	sb.WriteString(addr.Mailbox)
	sb.WriteString("@")
	sb.WriteString(addr.Host)
	sb.WriteString(">")

	return sb.String()
}

// Helper to get Date header as time.Time
func getDateHeader(msg *message) time.Time {
	env := msg.envelope()
	if env != nil && !env.Date.IsZero() {
		return env.Date
	}

	// Try to parse from headers
	r := msg.reader()
	if r != nil {
		fields := r.Header.Fields()
		for fields.Next() {
			if strings.EqualFold(fields.Key(), "Date") {
				value, _ := fields.Text()
				t, err := time.Parse(time.RFC1123Z, value)
				if err == nil {
					return t
				}
			}
		}
	}

	// Fallback to internal date
	return msg.t
}

// Helper to check if a NumSet is empty
func isNumSetEmpty(numSet imap.NumSet) bool {
	switch numSet := numSet.(type) {
	case imap.SeqSet:
		return len(numSet) == 0
	case imap.UIDSet:
		return len(numSet) == 0
	default:
		return true
	}
}
