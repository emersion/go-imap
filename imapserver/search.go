package imapserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleSearch(tag string, dec *imapwire.Decoder, numKind NumKind) error {
	if !dec.ExpectSP() {
		return dec.Err()
	}
	var (
		atom     string
		options  imap.SearchOptions
		extended bool
	)
	if maybeReadSearchKeyAtom(dec, &atom) && strings.EqualFold(atom, "RETURN") {
		if err := readSearchReturnOpts(dec, &options); err != nil {
			return fmt.Errorf("in search-return-opts: %w", err)
		}
		if !dec.ExpectSP() {
			return dec.Err()
		}
		extended = true
		atom = ""
		maybeReadSearchKeyAtom(dec, &atom)
	}
	if strings.EqualFold(atom, "CHARSET") {
		var charset string
		if !dec.ExpectSP() || !dec.ExpectAString(&charset) || !dec.ExpectSP() {
			return dec.Err()
		}
		switch strings.ToUpper(charset) {
		case "US-ASCII", "UTF-8":
			// nothing to do
		default:
			return &imap.Error{
				Type: imap.StatusResponseTypeNo,
				Code: imap.ResponseCodeBadCharset, // TODO: return list of supported charsets
				Text: "Only US-ASCII and UTF-8 are supported SEARCH charsets",
			}
		}
		atom = ""
		maybeReadSearchKeyAtom(dec, &atom)
	}

	var criteria imap.SearchCriteria
	for {
		var err error
		if atom != "" {
			err = readSearchKeyWithAtom(&criteria, dec, atom)
			atom = ""
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

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	// If no return option is specified, ALL is assumed
	if !options.ReturnMin && !options.ReturnMax && !options.ReturnAll && !options.ReturnCount {
		options.ReturnAll = true
	}

	data, err := c.session.Search(numKind, &criteria, &options)
	if err != nil {
		return err
	}

	if c.enabled.Has(imap.CapIMAP4rev2) || extended {
		return c.writeESearch(tag, data, &options)
	} else {
		return c.writeSearch(data.All)
	}
}

func (c *Conn) writeESearch(tag string, data *imap.SearchData, options *imap.SearchOptions) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom("*").SP().Atom("ESEARCH")
	if tag != "" {
		enc.SP().Special('(').Atom("TAG").SP().Atom(tag).Special(')')
	}
	if data.UID {
		enc.SP().Atom("UID")
	}
	// When there is no result, we need to send an ESEARCH response with no ALL
	// keyword
	if options.ReturnAll && !isNumSetEmpty(data.All) {
		enc.SP().Atom("ALL").SP().NumSet(data.All)
	}
	if options.ReturnMin && data.Min > 0 {
		enc.SP().Atom("MIN").SP().Number(data.Min)
	}
	if options.ReturnMax && data.Max > 0 {
		enc.SP().Atom("MAX").SP().Number(data.Max)
	}
	if options.ReturnCount {
		enc.SP().Atom("COUNT").SP().Number(data.Count)
	}
	return enc.CRLF()
}

func isNumSetEmpty(numSet imap.NumSet) bool {
	switch numSet := numSet.(type) {
	case imap.SeqSet:
		return len(numSet) == 0
	case imap.UIDSet:
		return len(numSet) == 0
	default:
		panic("unknown imap.NumSet type")
	}
}

func (c *Conn) writeSearch(numSet imap.NumSet) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom("*").SP().Atom("SEARCH")
	var ok bool
	switch numSet := numSet.(type) {
	case imap.SeqSet:
		var nums []uint32
		nums, ok = numSet.Nums()
		for _, num := range nums {
			enc.SP().Number(num)
		}
	case imap.UIDSet:
		var uids []imap.UID
		uids, ok = numSet.Nums()
		for _, uid := range uids {
			enc.SP().UID(uid)
		}
	}
	if !ok {
		return fmt.Errorf("imapserver: failed to enumerate message numbers in SEARCH response")
	}
	return enc.CRLF()
}

func readSearchReturnOpts(dec *imapwire.Decoder, options *imap.SearchOptions) error {
	if !dec.ExpectSP() {
		return dec.Err()
	}
	return dec.ExpectList(func() error {
		var name string
		if !dec.ExpectAtom(&name) {
			return dec.Err()
		}
		switch strings.ToUpper(name) {
		case "MIN":
			options.ReturnMin = true
		case "MAX":
			options.ReturnMax = true
		case "ALL":
			options.ReturnAll = true
		case "COUNT":
			options.ReturnCount = true
		default:
			return newClientBugError("unknown SEARCH RETURN option")
		}
		return nil
	})
}

func maybeReadSearchKeyAtom(dec *imapwire.Decoder, ptr *string) bool {
	return dec.Func(ptr, func(ch byte) bool {
		return ch == '*' || imapwire.IsAtomChar(ch)
	})
}

func readSearchKey(criteria *imap.SearchCriteria, dec *imapwire.Decoder) error {
	var key string
	if maybeReadSearchKeyAtom(dec, &key) {
		return readSearchKeyWithAtom(criteria, dec, key)
	}
	return dec.ExpectList(func() error {
		return readSearchKey(criteria, dec)
	})
}

func readSearchKeyWithAtom(criteria *imap.SearchCriteria, dec *imapwire.Decoder, key string) error {
	key = strings.ToUpper(key)
	switch key {
	case "ALL":
		// nothing to do
	case "UID":
		var uidSet imap.UIDSet
		if !dec.ExpectSP() || !dec.ExpectUIDSet(&uidSet) {
			return dec.Err()
		}
		criteria.UID = append(criteria.UID, uidSet)
	case "ANSWERED", "DELETED", "DRAFT", "FLAGGED", "RECENT", "SEEN":
		criteria.Flag = append(criteria.Flag, searchKeyFlag(key))
	case "UNANSWERED", "UNDELETED", "UNDRAFT", "UNFLAGGED", "UNSEEN":
		notKey := strings.TrimPrefix(key, "UN")
		criteria.NotFlag = append(criteria.NotFlag, searchKeyFlag(notKey))
	case "NEW":
		criteria.Flag = append(criteria.Flag, internal.FlagRecent)
		criteria.NotFlag = append(criteria.Flag, imap.FlagSeen)
	case "OLD":
		criteria.NotFlag = append(criteria.NotFlag, internal.FlagRecent)
	case "KEYWORD", "UNKEYWORD":
		if !dec.ExpectSP() {
			return dec.Err()
		}
		flag, err := internal.ExpectFlag(dec)
		if err != nil {
			return err
		}
		switch key {
		case "KEYWORD":
			criteria.Flag = append(criteria.Flag, flag)
		case "UNKEYWORD":
			criteria.NotFlag = append(criteria.NotFlag, flag)
		}
	case "BCC", "CC", "FROM", "SUBJECT", "TO":
		var value string
		if !dec.ExpectSP() || !dec.ExpectAString(&value) {
			return dec.Err()
		}
		criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{
			Key:   strings.Title(strings.ToLower(key)),
			Value: value,
		})
	case "HEADER":
		var key, value string
		if !dec.ExpectSP() || !dec.ExpectAString(&key) || !dec.ExpectSP() || !dec.ExpectAString(&value) {
			return dec.Err()
		}
		criteria.Header = append(criteria.Header, imap.SearchCriteriaHeaderField{
			Key:   key,
			Value: value,
		})
	case "SINCE", "BEFORE", "ON", "SENTSINCE", "SENTBEFORE", "SENTON":
		if !dec.ExpectSP() {
			return dec.Err()
		}
		t, err := internal.ExpectDate(dec)
		if err != nil {
			return err
		}
		var dateCriteria imap.SearchCriteria
		switch key {
		case "SINCE":
			dateCriteria.Since = t
		case "BEFORE":
			dateCriteria.Before = t
		case "ON":
			dateCriteria.Since = t
			dateCriteria.Before = t.Add(24 * time.Hour)
		case "SENTSINCE":
			dateCriteria.SentSince = t
		case "SENTBEFORE":
			dateCriteria.SentBefore = t
		case "SENTON":
			dateCriteria.SentSince = t
			dateCriteria.SentBefore = t.Add(24 * time.Hour)
		}
		criteria.And(&dateCriteria)
	case "BODY":
		var body string
		if !dec.ExpectSP() || !dec.ExpectAString(&body) {
			return dec.Err()
		}
		criteria.Body = append(criteria.Body, body)
	case "TEXT":
		var text string
		if !dec.ExpectSP() || !dec.ExpectAString(&text) {
			return dec.Err()
		}
		criteria.Text = append(criteria.Text, text)
	case "LARGER", "SMALLER":
		var n int64
		if !dec.ExpectSP() || !dec.ExpectNumber64(&n) {
			return dec.Err()
		}
		switch key {
		case "LARGER":
			criteria.And(&imap.SearchCriteria{Larger: n})
		case "SMALLER":
			criteria.And(&imap.SearchCriteria{Smaller: n})
		}
	case "NOT":
		if !dec.ExpectSP() {
			return dec.Err()
		}
		var not imap.SearchCriteria
		if err := readSearchKey(&not, dec); err != nil {
			return nil
		}
		criteria.Not = append(criteria.Not, not)
	case "OR":
		if !dec.ExpectSP() {
			return dec.Err()
		}
		var or [2]imap.SearchCriteria
		if err := readSearchKey(&or[0], dec); err != nil {
			return nil
		}
		if !dec.ExpectSP() {
			return dec.Err()
		}
		if err := readSearchKey(&or[1], dec); err != nil {
			return nil
		}
		criteria.Or = append(criteria.Or, or)
	default:
		seqSet, err := imapwire.ParseSeqSet(key)
		if err != nil {
			return err
		}
		criteria.SeqNum = append(criteria.SeqNum, seqSet)
	}
	return nil
}

func searchKeyFlag(key string) imap.Flag {
	return imap.Flag("\\" + strings.Title(strings.ToLower(key)))
}
