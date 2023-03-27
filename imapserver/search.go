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
		atom    string
		options imap.SearchOptions
	)
	if dec.Atom(&atom) && atom == "RETURN" {
		var err error
		options.Return, err = readSearchReturnOpts(dec)
		if err != nil {
			return fmt.Errorf("in search-return-opts: %w", err)
		}
		if !dec.ExpectSP() {
			return dec.Err()
		}
		atom = ""
		dec.Atom(&atom)
	}
	if atom == "CHARSET" {
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
		dec.Atom(&atom)
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

	data, err := c.session.Search(numKind, &criteria, &options)
	if err != nil {
		return err
	}
	// TODO: write obsolete search response for IMAP4rev1 clients
	return c.writeESearch(tag, data, &options)
}

func (c *Conn) writeESearch(tag string, data *imap.SearchData, options *imap.SearchOptions) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	returnOpts := make(map[imap.SearchReturnOption]bool)
	for _, opt := range options.Return {
		returnOpts[opt] = true
	}

	enc.Atom("*").SP().Atom("ESEARCH")
	if tag != "" {
		enc.SP().Special('(').Atom("TAG").SP().Atom(tag).Special(')')
	}
	if data.UID {
		enc.SP().Atom("UID")
	}
	if returnOpts[imap.SearchReturnAll] || len(options.Return) == 0 {
		enc.SP().Atom("ALL").SP().Atom(data.All.String())
	}
	if returnOpts[imap.SearchReturnMin] {
		enc.SP().Atom("MIN").SP().Number(data.Min)
	}
	if returnOpts[imap.SearchReturnMax] {
		enc.SP().Atom("MAX").SP().Number(data.Max)
	}
	if returnOpts[imap.SearchReturnCount] {
		enc.SP().Atom("COUNT").SP().Number(data.Count)
	}
	return enc.CRLF()
}

func readSearchReturnOpts(dec *imapwire.Decoder) ([]imap.SearchReturnOption, error) {
	if !dec.ExpectSP() {
		return nil, dec.Err()
	}
	var l []imap.SearchReturnOption
	err := dec.ExpectList(func() error {
		var name string
		if !dec.ExpectAtom(&name) {
			return dec.Err()
		}
		switch opt := imap.SearchReturnOption(name); opt {
		case imap.SearchReturnMin, imap.SearchReturnMax, imap.SearchReturnAll, imap.SearchReturnCount:
			l = append(l, opt)
		default:
			return newClientBugError("unknown SEARCH RETURN option")
		}
		return nil
	})
	return l, err
}

func readSearchKey(criteria *imap.SearchCriteria, dec *imapwire.Decoder) error {
	var key string
	if dec.Atom(&key) {
		return readSearchKeyWithAtom(criteria, dec, key)
	}
	return dec.ExpectList(func() error {
		return readSearchKey(criteria, dec)
	})
}

func readSearchKeyWithAtom(criteria *imap.SearchCriteria, dec *imapwire.Decoder, key string) error {
	switch key {
	case "ALL":
		// nothing to do
	case "UID":
		var seqSetStr string
		if !dec.ExpectSP() || !dec.ExpectAtom(&seqSetStr) {
			return dec.Err()
		}
		seqSet, err := imap.ParseSeqSet(seqSetStr)
		if err != nil {
			return err
		}
		criteria.UID = seqSet // TODO: intersect
	case "ANSWERED", "DELETED", "DRAFT", "FLAGGED", "RECENT", "SEEN":
		criteria.Flag = append(criteria.Flag, searchKeyFlag(key))
	case "UNANSWERED", "UNDELETED", "UNDRAFT", "UNFLAGGED", "UNSEEN":
		notKey := strings.TrimPrefix(key, "UN")
		criteria.NotFlag = append(criteria.NotFlag, searchKeyFlag(notKey))
	case "NEW":
		criteria.Flag = append(criteria.Flag, imap.FlagRecent)
		criteria.NotFlag = append(criteria.Flag, imap.FlagSeen)
	case "OLD":
		criteria.NotFlag = append(criteria.NotFlag, imap.FlagRecent)
	case "KEYWORD", "UNKEYWORD":
		if !dec.ExpectSP() {
			return dec.Err()
		}
		flagStr, err := internal.ReadFlag(dec)
		if err != nil {
			return err
		}
		flag := imap.Flag(flagStr)
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
		switch key {
		case "SINCE":
			criteria.Since = intersectSince(criteria.Since, t)
		case "BEFORE":
			criteria.Before = intersectBefore(criteria.Before, t)
		case "ON":
			criteria.Since = intersectSince(criteria.Since, t)
			criteria.Before = intersectBefore(criteria.Before, t.Add(24*time.Hour))
		case "SENTSINCE":
			criteria.SentSince = intersectSince(criteria.SentSince, t)
		case "SENTBEFORE":
			criteria.SentBefore = intersectBefore(criteria.SentBefore, t)
		case "SENTON":
			criteria.SentSince = intersectSince(criteria.SentSince, t)
			criteria.SentBefore = intersectBefore(criteria.SentBefore, t.Add(24*time.Hour))
		}
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
			if criteria.Larger == 0 || n > criteria.Larger {
				criteria.Larger = n
			}
		case "SMALLER":
			if criteria.Smaller == 0 || n < criteria.Smaller {
				criteria.Smaller = n
			}
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
		seqSet, err := imap.ParseSeqSet(key)
		if err != nil {
			return err
		}
		criteria.SeqNum = seqSet // TODO: intersect
	}
	return nil
}

func searchKeyFlag(key string) imap.Flag {
	return imap.Flag("\\" + strings.Title(strings.ToLower(key)))
}

func intersectSince(t1, t2 time.Time) time.Time {
	switch {
	case t1.IsZero():
		return t2
	case t2.IsZero():
		return t1
	case t1.After(t2):
		return t1
	default:
		return t2
	}
}

func intersectBefore(t1, t2 time.Time) time.Time {
	switch {
	case t1.IsZero():
		return t2
	case t2.IsZero():
		return t1
	case t1.Before(t2):
		return t1
	default:
		return t2
	}
}
