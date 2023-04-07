package imapclient

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func returnSearchOptions(options *imap.SearchOptions) []string {
	if options == nil {
		return nil
	}

	m := map[string]bool{
		"MIN":   options.ReturnMin,
		"MAX":   options.ReturnMax,
		"ALL":   options.ReturnAll,
		"COUNT": options.ReturnCount,
	}

	var l []string
	for k, ret := range m {
		if ret {
			l = append(l, k)
		}
	}
	return l
}

func (c *Client) search(uid bool, criteria *imap.SearchCriteria, options *imap.SearchOptions) *SearchCommand {
	// TODO: add support for SEARCHRES

	// The IMAP4rev2 SEARCH charset defaults to UTF-8. For IMAP4rev1 the
	// default is undefined and only US-ASCII support is required. What's more,
	// some servers completely reject the CHARSET keyword. So, let's check if
	// we actually have UTF-8 strings in the search criteria before using that.
	// TODO: there might be a benefit in specifying CHARSET UTF-8 for IMAP4rev1
	// servers even if we only send ASCII characters: the server then must
	// decode encoded headers and Content-Transfer-Encoding before matching the
	// criteria.
	var charset string
	if !c.Caps().Has(imap.CapIMAP4rev2) && !searchCriteriaIsASCII(criteria) {
		charset = "UTF-8"
	}

	cmd := &SearchCommand{}
	enc := c.beginCommand(uidCmdName("SEARCH", uid), cmd)
	if returnOpts := returnSearchOptions(options); len(returnOpts) > 0 {
		enc.SP().Atom("RETURN").SP().List(len(returnOpts), func(i int) {
			enc.Atom(returnOpts[i])
		})
	}
	enc.SP()
	if charset != "" {
		enc.Atom("CHARSET").SP().Atom(charset).SP()
	}
	writeSearchKey(enc.Encoder, criteria)
	enc.end()
	return cmd
}

// Search sends a SEARCH command.
func (c *Client) Search(criteria *imap.SearchCriteria, options *imap.SearchOptions) *SearchCommand {
	return c.search(false, criteria, options)
}

// UIDSearch sends a UID SEARCH command.
func (c *Client) UIDSearch(criteria *imap.SearchCriteria, options *imap.SearchOptions) *SearchCommand {
	return c.search(true, criteria, options)
}

func (c *Client) handleSearch() error {
	cmd := findPendingCmdByType[*SearchCommand](c)
	for c.dec.SP() {
		var num uint32
		if !c.dec.ExpectNumber(&num) {
			return c.dec.Err()
		}
		if cmd != nil {
			cmd.data.All.AddNum(num)
		}
	}
	return nil
}

func (c *Client) handleESearch() error {
	if !c.dec.ExpectSP() {
		return c.dec.Err()
	}
	tag, data, err := readESearchResponse(c.dec)
	if err != nil {
		return err
	}
	cmd := c.findPendingCmdFunc(func(anyCmd command) bool {
		cmd, ok := anyCmd.(*SearchCommand)
		if !ok {
			return false
		}
		if tag != "" {
			return cmd.tag == tag
		} else {
			return true
		}
	})
	if cmd != nil {
		cmd := cmd.(*SearchCommand)
		cmd.data = *data
	}
	return nil
}

// SearchCommand is a SEARCH command.
type SearchCommand struct {
	cmd
	data imap.SearchData
}

func (cmd *SearchCommand) Wait() (*imap.SearchData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

func writeSearchKey(enc *imapwire.Encoder, criteria *imap.SearchCriteria) {
	enc.Special('(')

	firstItem := true
	encodeItem := func(s string) *imapwire.Encoder {
		if !firstItem {
			enc.SP()
		}
		firstItem = false
		return enc.Atom(s)
	}

	for _, seqSet := range criteria.SeqNum {
		encodeItem(seqSet.String())
	}
	for _, seqSet := range criteria.UID {
		encodeItem("UID").SP().SeqSet(seqSet)
	}

	if !criteria.Since.IsZero() && !criteria.Before.IsZero() && criteria.Before.Sub(criteria.Since) == 24*time.Hour {
		encodeItem("ON").SP().String(criteria.Since.Format(internal.DateLayout))
	} else {
		if !criteria.Since.IsZero() {
			encodeItem("SINCE").SP().String(criteria.Since.Format(internal.DateLayout))
		}
		if !criteria.Before.IsZero() {
			encodeItem("BEFORE").SP().String(criteria.Before.Format(internal.DateLayout))
		}
	}
	if !criteria.SentSince.IsZero() && !criteria.SentBefore.IsZero() && criteria.SentBefore.Sub(criteria.SentSince) == 24*time.Hour {
		encodeItem("SENTON").SP().String(criteria.SentSince.Format(internal.DateLayout))
	} else {
		if !criteria.SentSince.IsZero() {
			encodeItem("SENTSINCE").SP().String(criteria.SentSince.Format(internal.DateLayout))
		}
		if !criteria.SentBefore.IsZero() {
			encodeItem("SENTBEFORE").SP().String(criteria.SentBefore.Format(internal.DateLayout))
		}
	}

	for _, kv := range criteria.Header {
		switch k := strings.ToUpper(kv.Key); k {
		case "BCC", "CC", "FROM", "SUBJECT", "TO":
			encodeItem(k)
		default:
			encodeItem("HEADER").SP().String(kv.Key)
		}
		enc.SP().String(kv.Value)
	}

	for _, s := range criteria.Body {
		encodeItem("BODY").SP().String(s)
	}
	for _, s := range criteria.Text {
		encodeItem("TEXT").SP().String(s)
	}

	for _, flag := range criteria.Flag {
		if k := flagSearchKey(flag); k != "" {
			encodeItem(k)
		} else {
			encodeItem("KEYWORD").SP().Flag(flag)
		}
	}
	for _, flag := range criteria.NotFlag {
		if k := flagSearchKey(flag); k != "" {
			encodeItem("UN" + k)
		} else {
			encodeItem("UNKEYWORD").SP().Flag(flag)
		}
	}

	if criteria.Larger > 0 {
		encodeItem("LARGER").SP().Number64(criteria.Larger)
	}
	if criteria.Smaller > 0 {
		encodeItem("SMALLER").SP().Number64(criteria.Smaller)
	}

	if criteria.Not != nil {
		encodeItem("NOT").SP()
		writeSearchKey(enc, criteria.Not)
	}
	for _, or := range criteria.Or {
		encodeItem("OR").SP()
		writeSearchKey(enc, &or[0])
		enc.SP()
		writeSearchKey(enc, &or[1])
	}

	if firstItem {
		enc.Atom("ALL")
	}

	enc.Special(')')
}

func flagSearchKey(flag imap.Flag) string {
	switch flag {
	case imap.FlagAnswered, imap.FlagDeleted, imap.FlagDraft, imap.FlagFlagged, imap.FlagSeen:
		return strings.ToUpper(strings.TrimPrefix(string(flag), "\\"))
	default:
		return ""
	}
}

func readESearchResponse(dec *imapwire.Decoder) (tag string, data *imap.SearchData, err error) {
	data = &imap.SearchData{}

	if dec.Special('(') { // search-correlator
		var correlator string
		if !dec.ExpectAtom(&correlator) || !dec.ExpectSP() || !dec.ExpectAString(&tag) || !dec.ExpectSpecial(')') || !dec.ExpectSP() {
			return "", nil, dec.Err()
		}
		if correlator != "TAG" {
			return "", nil, fmt.Errorf("in search-correlator: name must be TAG, but got %q", correlator)
		}
	}

	var name string
	if !dec.ExpectAtom(&name) || !dec.ExpectSP() {
		return "", nil, dec.Err()
	}
	data.UID = name == "UID"
	if data.UID {
		if !dec.ExpectAtom(&name) || !dec.ExpectSP() {
			return "", nil, dec.Err()
		}
	}
	for {
		switch strings.ToUpper(name) {
		case "MIN":
			var num uint32
			if !dec.ExpectNumber(&num) {
				return "", nil, dec.Err()
			}
			data.Min = num
		case "MAX":
			var num uint32
			if !dec.ExpectNumber(&num) {
				return "", nil, dec.Err()
			}
			data.Max = num
		case "ALL":
			if !dec.ExpectSeqSet(&data.All) {
				return "", nil, dec.Err()
			}
		case "COUNT":
			var num uint32
			if !dec.ExpectNumber(&num) {
				return "", nil, dec.Err()
			}
			data.Count = num
		default:
			if !dec.DiscardValue() {
				return "", nil, dec.Err()
			}
		}

		if !dec.SP() {
			break
		}

		if !dec.ExpectAtom(&name) || !dec.ExpectSP() {
			return "", nil, dec.Err()
		}
	}

	return tag, data, nil
}

func searchCriteriaIsASCII(criteria *imap.SearchCriteria) bool {
	for _, kv := range criteria.Header {
		if !isASCII(kv.Key) || !isASCII(kv.Value) {
			return false
		}
	}
	for _, s := range criteria.Body {
		if !isASCII(s) {
			return false
		}
	}
	for _, s := range criteria.Text {
		if !isASCII(s) {
			return false
		}
	}
	if criteria.Not != nil && !searchCriteriaIsASCII(criteria.Not) {
		return false
	}
	for _, or := range criteria.Or {
		if !searchCriteriaIsASCII(&or[0]) || !searchCriteriaIsASCII(&or[1]) {
			return false
		}
	}
	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
