package imapserver

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// SessionSort is an IMAP session which supports SORT.
type SessionSort interface {
	Session

	// Sort performs a sort operation with the provided criteria.
	Sort(kind NumKind, sortCriteria []imap.SortCriterion, criteria *imap.SearchCriteria, options *imap.SearchOptions, sortOptions *imap.SortOptions) (*imap.SearchData, error)
}

func (c *Conn) handleSort(tag string, dec *imapwire.Decoder, numKind NumKind) error {
	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	// Parse sort criteria
	var sortCriteria []imap.SortCriterion
	var displayLocale string

	if !dec.ExpectSP() {
		return dec.Err()
	}

	// Check for DISPLAY parameter (SORT=DISPLAY extension)
	// We'll check for DISPLAY as an optional parameter
	hasInitialCriterion := false
	var atom string

	if dec.Atom(&atom) {
		if strings.ToUpper(atom) == "DISPLAY" {
			// DISPLAY parameter found
			if !dec.ExpectSP() {
				return dec.Err()
			}

			// Parse the locale
			if !dec.ExpectAString(&displayLocale) || !dec.ExpectSP() {
				return dec.Err()
			}
		} else {
			// Not a DISPLAY parameter, must be the first sort criterion
			hasInitialCriterion = true
			// We'll handle it as part of the list
		}
	}

	// Parse the list of sort criteria
	err := dec.ExpectList(func() error {
		// If we already read the first atom and it wasn't DISPLAY, use it as the first criterion
		if hasInitialCriterion {
			var criterion imap.SortCriterion
			if atom == "REVERSE" {
				criterion.Reverse = true
				if !dec.ExpectSP() || !dec.ExpectAtom(&atom) {
					return dec.Err()
				}
			}
			criterion.Key = imap.SortKey(atom)
			sortCriteria = append(sortCriteria, criterion)
			hasInitialCriterion = false
			return nil
		}
		var atom string
		if !dec.ExpectAtom(&atom) {
			return dec.Err()
		}

		var criterion imap.SortCriterion
		if atom == "REVERSE" {
			criterion.Reverse = true
			if !dec.ExpectSP() || !dec.ExpectAtom(&atom) {
				return dec.Err()
			}
		}
		criterion.Key = imap.SortKey(atom)
		sortCriteria = append(sortCriteria, criterion)
		return nil
	})

	if err != nil {
		return err
	}

	// Parse "UTF-8" (required according to RFC 5256)
	var charsetAtom string
	if !dec.ExpectSP() || !dec.ExpectAtom(&charsetAtom) {
		return dec.Err()
	}
	if charsetAtom != "UTF-8" {
		return &imap.Error{
			Type: imap.StatusResponseTypeBad,
			Text: "Charset must be UTF-8",
		}
	}

	// Parse search criteria
	var criteria imap.SearchCriteria
	var searchOptions imap.SearchOptions
	if !dec.ExpectSP() {
		return dec.Err()
	}

	// Use the search key parser from search.go
	var searchAtom string
	if maybeReadSearchKeyAtom(dec, &searchAtom) && strings.EqualFold(searchAtom, "RETURN") {
		if err := readSearchReturnOpts(dec, &searchOptions); err != nil {
			return fmt.Errorf("in search-return-opts: %w", err)
		}
		if !dec.ExpectSP() {
			return dec.Err()
		}
		searchAtom = ""
		maybeReadSearchKeyAtom(dec, &searchAtom)
	}

	for {
		var err error
		if searchAtom != "" {
			err = readSearchKeyWithAtom(&criteria, dec, searchAtom)
			searchAtom = ""
		} else {
			err = readSearchKey(&criteria, dec)
		}
		if err != nil {
			return fmt.Errorf("in search-key: %w", err)
		}

		if !dec.SP() {
			break
		}
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	// Execute sort
	sortSession, ok := c.session.(SessionSort)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Text: "SORT not implemented",
		}
	}

	// Create sort options with display locale if specified
	sortOptions := &imap.SortOptions{
		DisplayLocale: displayLocale,
	}

	data, err := sortSession.Sort(numKind, sortCriteria, &criteria, &searchOptions, sortOptions)
	if err != nil {
		return err
	}

	// Write sort result
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom("*").SP().Atom("SORT")

	// Output the numbers based on the numKind
	switch numSet := data.All.(type) {
	case imap.SeqSet:
		nums, ok := numSet.Nums()
		if ok {
			for _, num := range nums {
				enc.SP().Number(num)
			}
		}
	case imap.UIDSet:
		uids, ok := numSet.Nums()
		if ok {
			for _, uid := range uids {
				enc.SP().UID(uid)
			}
		}
	}

	err = enc.CRLF()
	if err != nil {
		return err
	}

	// If this is an extended search, we need to send an ESEARCH response
	if c.enabled.Has(imap.CapIMAP4rev2) || searchOptions.ReturnMin ||
		searchOptions.ReturnMax || searchOptions.ReturnAll || searchOptions.ReturnCount {
		return c.writeESearch(tag, data, &searchOptions, numKind)
	}

	return nil
}
