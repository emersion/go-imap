package imapserver

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

type SortData struct {
	Nums  []uint32
	Min   uint32
	Max   uint32
	Count uint32
}

type SessionSort interface {
	Session

	Sort(numKind NumKind, criteria *imap.SearchCriteria, sortCriteria []imap.SortCriterion) ([]uint32, error)
}

type esortReturnOptions struct {
	Min   bool
	Max   bool
	Count bool
	All   bool
}

func (c *Conn) handleSort(tag string, dec *imapwire.Decoder, numKind NumKind) error {
	if !dec.ExpectSP() {
		return dec.Err()
	}

	var esortReturnOpts esortReturnOptions
	esortReturnOpts.All = true // Default if no RETURN or RETURN (ALL)

	var atom string
	// dec.Func returns true if an atom is read; 'atom' will contain it.
	// If the next token is not an atom (e.g., '('), it returns false and dec.Err() is nil.
	if dec.Func(&atom, imapwire.IsAtomChar) && strings.EqualFold(atom, "RETURN") {
		// Atom "RETURN" was successfully read and consumed.
		if !dec.ExpectSP() {
			return dec.Err()
		}

		esortReturnOpts.All = false // Explicit RETURN given, so default ALL is off unless specified in list

		parseReturnErr := dec.ExpectList(func() error {
			var opt string
			if !dec.ExpectAtom(&opt) {
				return dec.Err()
			}
			opt = strings.ToUpper(opt)
			switch opt {
			case "MIN":
				esortReturnOpts.Min = true
			case "MAX":
				esortReturnOpts.Max = true
			case "COUNT":
				esortReturnOpts.Count = true
			case "ALL":
				esortReturnOpts.All = true
			default:
				// RFC 5267: Servers MUST ignore any unknown sort-return-opt.
			}
			return nil
		})
		if parseReturnErr != nil {
			return parseReturnErr
		}

		if esortReturnOpts.All && (esortReturnOpts.Min || esortReturnOpts.Max || esortReturnOpts.Count) {
			return &imap.Error{
				Type: imap.StatusResponseTypeBad,
				Text: "ESORT RETURN ALL cannot be combined with MIN, MAX, or COUNT",
			}
		}

		// If RETURN was specified but resulted in no recognized options, default to ALL.
		// This means if RETURN () or RETURN (UNKNOWN_OPT) is sent, it behaves as if RETURN (ALL) or no RETURN was sent.
		if !esortReturnOpts.Min && !esortReturnOpts.Max && !esortReturnOpts.Count && !esortReturnOpts.All {
			esortReturnOpts.All = true
		}

		if !dec.ExpectSP() { // Expect SP after RETURN (...)
			return dec.Err()
		}
	} else if dec.Err() != nil {
		// dec.Func failed for a reason other than the first char not matching (e.g. EOF or malformed atom)
		return dec.Err()
	}

	var sortCriteria []imap.SortCriterion
	listErr := dec.ExpectList(func() error {
		for { // Loop to correctly parse multiple sort criteria items
			var criterion imap.SortCriterion
			var atom string
			if !dec.ExpectAtom(&atom) {
				return dec.Err()
			}

			if strings.EqualFold(atom, "REVERSE") {
				criterion.Reverse = true
				if !dec.ExpectSP() || !dec.ExpectAtom(&atom) {
					return dec.Err()
				}
			}

			criterion.Key = imap.SortKey(strings.ToUpper(atom))
			sortCriteria = append(sortCriteria, criterion)

			if !dec.SP() { // If no more SP, then no more sort criteria in this list
				break
			}
		}
		return nil
	})
	if listErr != nil {
		return listErr
	}

	// Parse charset - must be UTF-8 for SORT
	if !dec.ExpectSP() {
		return dec.Err()
	}
	var charset string
	if !dec.ExpectAtom(&charset) || !dec.ExpectSP() {
		return dec.Err()
	}
	if !strings.EqualFold(charset, "UTF-8") {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeBadCharset,
			Text: "Only UTF-8 is supported for SORT",
		}
	}

	// Parse search criteria
	var criteria imap.SearchCriteria
	for {
		if err := readSearchKey(&criteria, dec); err != nil {
			return fmt.Errorf("in search-key: %w", err)
		}
		if !dec.SP() { // If no more SP, then no more search keys
			break
		}
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	var sortedNums []uint32
	if sortSession, ok := c.session.(SessionSort); ok {
		var sortErr error
		sortedNums, sortErr = sortSession.Sort(numKind, &criteria, sortCriteria)
		if sortErr != nil {
			return sortErr
		}
	} else {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeCannot,
			Text: "SORT command is not supported by this session",
		}
	}

	data := &SortData{Nums: sortedNums}
	if len(sortedNums) > 0 {
		data.Count = uint32(len(sortedNums))
		min, max := sortedNums[0], sortedNums[0]
		for _, num := range sortedNums {
			if num < min {
				min = num
			}
			if num > max {
				max = num
			}
		}
		data.Min = min
		data.Max = max
	}

	return c.writeSortResponse(tag, numKind, data, &esortReturnOpts)
}

func (c *Conn) writeSortResponse(tag string, numKind NumKind, data *SortData, returnOpts *esortReturnOptions) error {
	// For ESORT, if RETURN options other than ALL are specified, send an ESEARCH response.
	// See RFC 5267 section 4.2.
	if c.server.options.caps().Has(imap.CapESort) && !returnOpts.All {
		enc := newResponseEncoder(c)
		enc.Atom("*").SP().Atom("ESEARCH")
		if tag != "" {
			enc.SP().Special('(').Atom("TAG").SP().String(tag).Special(')')
		}
		if numKind == NumKindUID {
			enc.SP().Atom("UID")
		}
		if returnOpts.Min {
			if data.Count > 0 {
				enc.SP().Atom("MIN").SP().Number(data.Min)
			}
		}
		if returnOpts.Max {
			if data.Count > 0 {
				enc.SP().Atom("MAX").SP().Number(data.Max)
			}
		}
		if returnOpts.Count {
			enc.SP().Atom("COUNT").SP().Number(data.Count)
		}
		if err := enc.CRLF(); err != nil {
			enc.end()
			return err
		}
		enc.end()
	}

	// A SORT response is always sent, either for a regular SORT, or following
	// an ESEARCH response for an ESORT.
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("SORT")
	for _, num := range data.Nums {
		enc.SP().Number(num)
	}
	return enc.CRLF()
}
