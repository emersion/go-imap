package imapmemserver

import (
	"bufio"
	"bytes"
	"net/mail"
	"net/textproto"
	"sort"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

// Sort performs a SORT command.
func (mbox *MailboxView) Sort(numKind imapserver.NumKind, criteria *imap.SearchCriteria, sortCriteria []imap.SortCriterion) ([]uint32, error) {
	mbox.mutex.Lock()
	defer mbox.mutex.Unlock()

	// Apply search criteria
	mbox.staticSearchCriteria(criteria)

	// First find all messages that match the search criteria
	var matchedMessages []*message
	var matchedSeqNums []uint32
	var matchedIndices []int
	for i, msg := range mbox.l {
		seqNum := mbox.tracker.EncodeSeqNum(uint32(i) + 1)

		if !msg.search(seqNum, criteria) {
			continue
		}

		matchedMessages = append(matchedMessages, msg)
		matchedSeqNums = append(matchedSeqNums, seqNum)
		matchedIndices = append(matchedIndices, i)
	}

	// Sort the matched messages based on the sort criteria
	sortMatchedMessages(matchedMessages, matchedSeqNums, matchedIndices, sortCriteria)

	// Create sorted response
	var data []uint32
	for i, msg := range matchedMessages {
		var num uint32
		switch numKind {
		case imapserver.NumKindSeq:
			if matchedSeqNums[i] == 0 {
				continue
			}
			num = matchedSeqNums[i]
		case imapserver.NumKindUID:
			num = uint32(msg.uid)
		}
		data = append(data, num)
	}

	return data, nil
}

// sortMatchedMessages sorts messages according to the specified sort criteria
func sortMatchedMessages(messages []*message, seqNums []uint32, indices []int, criteria []imap.SortCriterion) {
	if len(messages) < 2 {
		return // Nothing to sort
	}

	// Create a slice of indices for sorting
	indices2 := make([]int, len(messages))
	for i := range indices2 {
		indices2[i] = i
	}

	// Sort the indices based on the criteria
	sort.SliceStable(indices2, func(i, j int) bool {
		i2, j2 := indices2[i], indices2[j]

		// Apply each criterion in order until we find a difference
		for _, criterion := range criteria {
			result := compareByCriterion(messages[i2], messages[j2], criterion.Key)

			// Apply reverse if needed
			if criterion.Reverse {
				result = -result
			}

			// If comparison yields a difference, return the result
			if result < 0 {
				return true
			} else if result > 0 {
				return false
			}
			// If equal, continue to the next criterion
		}

		// If all criteria are equal, maintain original order
		return i < j
	})

	// Reorder the original slices according to the sorted indices
	newMessages := make([]*message, len(messages))
	newSeqNums := make([]uint32, len(seqNums))
	newIndices := make([]int, len(indices))

	for i, idx := range indices2 {
		newMessages[i] = messages[idx]
		newSeqNums[i] = seqNums[idx]
		newIndices[i] = indices[idx]
	}

	// Copy sorted slices back to original slices
	copy(messages, newMessages)
	copy(seqNums, newSeqNums)
	copy(indices, newIndices)
}

// compareByCriterion compares two messages based on a single criterion
// returns -1 if a < b, 0 if a == b, 1 if a > b
func compareByCriterion(a, b *message, key imap.SortKey) int {
	switch key {
	case imap.SortKeyArrival:
		// For ARRIVAL, we use the UID as the arrival order
		if a.uid < b.uid {
			return -1
		} else if a.uid > b.uid {
			return 1
		}
		return 0

	case imap.SortKeyDate:
		// Compare internal date
		if a.t.Before(b.t) {
			return -1
		} else if a.t.After(b.t) {
			return 1
		}
		return 0

	case imap.SortKeySize:
		// Compare message sizes
		aSize := len(a.buf)
		bSize := len(b.buf)
		if aSize < bSize {
			return -1
		} else if aSize > bSize {
			return 1
		}
		return 0

	case imap.SortKeyFrom:
		// NOTE: A fully compliant implementation as per RFC 5256 would parse
		// the address and sort by mailbox, then host. This is a simplified
		// case-insensitive comparison of the full header value.
		fromA := getHeader(a.buf, "From")
		fromB := getHeader(b.buf, "From")
		return strings.Compare(strings.ToLower(fromA), strings.ToLower(fromB))

	case imap.SortKeyTo:
		// NOTE: Simplified comparison. See SortKeyFrom.
		toA := getHeader(a.buf, "To")
		toB := getHeader(b.buf, "To")
		return strings.Compare(strings.ToLower(toA), strings.ToLower(toB))

	case imap.SortKeyCc:
		// NOTE: Simplified comparison. See SortKeyFrom.
		ccA := getHeader(a.buf, "Cc")
		ccB := getHeader(b.buf, "Cc")
		return strings.Compare(strings.ToLower(ccA), strings.ToLower(ccB))

	case imap.SortKeySubject:
		// RFC 5256 specifies i;ascii-casemap collation, which is case-insensitive.
		subjA := getHeader(a.buf, "Subject")
		subjB := getHeader(b.buf, "Subject")
		return strings.Compare(strings.ToLower(subjA), strings.ToLower(subjB))

	case imap.SortKeyDisplay:
		// RFC 5957: sort by display-name, fallback to mailbox.
		fromA := getHeader(a.buf, "From")
		fromB := getHeader(b.buf, "From")

		addrA, errA := mail.ParseAddress(fromA)
		addrB, errB := mail.ParseAddress(fromB)

		var displayA, displayB string

		if errA == nil {
			if addrA.Name != "" {
				displayA = addrA.Name
			} else {
				displayA = addrA.Address
			}
		} else {
			displayA = fromA // Fallback to raw header on parse error
		}

		if errB == nil {
			if addrB.Name != "" {
				displayB = addrB.Name
			} else {
				displayB = addrB.Address
			}
		} else {
			displayB = fromB // Fallback to raw header on parse error
		}

		// A full implementation would use locale-aware sorting (e.g., golang.org/x/text/collate).
		// A case-insensitive comparison is a reasonable and significant improvement.
		return strings.Compare(strings.ToLower(displayA), strings.ToLower(displayB))

	default:
		// Default to no sorting for unknown criteria
		return 0
	}
}

// getHeader extracts a header value from a message's raw bytes.
// It performs a case-insensitive search for the key.
func getHeader(buf []byte, key string) string {
	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(buf)))
	hdr, err := r.ReadMIMEHeader()
	if err != nil {
		return "" // Or log the error
	}
	return hdr.Get(key)
}
