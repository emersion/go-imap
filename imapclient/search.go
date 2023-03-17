package imapclient

import (
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const searchDateLayout = "2-Jan-2006"

func (c *Client) search(uid bool, criteria *SearchCriteria) *SearchCommand {
	// TODO: result specifier, charset
	cmd := &SearchCommand{}
	enc := c.beginCommand(uidCmdName("SEARCH", uid), cmd)
	enc.SP()
	writeSearchKey(enc.Encoder, criteria)
	enc.end()
	return cmd
}

// Search sends a SEARCH command.
func (c *Client) Search(criteria *SearchCriteria) *SearchCommand {
	return c.search(false, criteria)
}

// UIDSearch sends a UID SEARCH command.
func (c *Client) UIDSearch(criteria *SearchCriteria) *SearchCommand {
	return c.search(true, criteria)
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

// SearchCommand is a SEARCH command.
type SearchCommand struct {
	cmd
	data SearchData
}

func (cmd *SearchCommand) Wait() (*SearchData, error) {
	return &cmd.data, cmd.cmd.Wait()
}

// SearchData is the data returned by a SEARCH command.
type SearchData struct {
	All imap.SeqSet
	// TODO: MIN, MAX, COUNT
}

// AllNums returns All as a slice of numbers.
func (data *SearchData) AllNums() []uint32 {
	// Note: a dynamic sequence set would be a server bug
	nums, _ := data.All.Nums()
	return nums
}

// SearchCriteria is a criteria for the SEARCH command.
//
// When multiple fields are populated, the result is the intersection ("and"
// function) of all messages that match the fields.
type SearchCriteria struct {
	SeqNum imap.SeqSet
	UID    imap.SeqSet

	// Only the date is used, the time and timezone are ignored
	Since      time.Time
	Before     time.Time
	SentSince  time.Time
	SentBefore time.Time

	Header []SearchCriteriaHeaderField
	Body   []string
	Text   []string

	Flag    []imap.Flag
	NotFlag []imap.Flag

	Larger  int64
	Smaller int64

	Not []SearchCriteria
	Or  [][2]SearchCriteria
}

type SearchCriteriaHeaderField struct {
	Key, Value string
}

func writeSearchKey(enc *imapwire.Encoder, criteria *SearchCriteria) {
	enc.Special('(')

	firstItem := true
	encodeItem := func(s string) *imapwire.Encoder {
		if !firstItem {
			enc.SP()
		}
		firstItem = false
		return enc.Atom(s)
	}

	if len(criteria.SeqNum) > 0 {
		encodeItem(criteria.SeqNum.String())
	}
	if len(criteria.UID) > 0 {
		encodeItem("UID").SP().Atom(criteria.UID.String())
	}

	if !criteria.Since.IsZero() && !criteria.Before.IsZero() && criteria.Before.Sub(criteria.Since) == 24*time.Hour {
		encodeItem("ON").SP().String(criteria.Since.Format(searchDateLayout))
	} else {
		if !criteria.Since.IsZero() {
			encodeItem("SINCE").SP().String(criteria.Since.Format(searchDateLayout))
		}
		if !criteria.Before.IsZero() {
			encodeItem("BEFORE").SP().String(criteria.Before.Format(searchDateLayout))
		}
	}
	if !criteria.SentSince.IsZero() && !criteria.SentBefore.IsZero() && criteria.SentBefore.Sub(criteria.SentSince) == 24*time.Hour {
		encodeItem("SENTON").SP().String(criteria.SentSince.Format(searchDateLayout))
	} else {
		if !criteria.SentSince.IsZero() {
			encodeItem("SENTSINCE").SP().String(criteria.SentSince.Format(searchDateLayout))
		}
		if !criteria.SentBefore.IsZero() {
			encodeItem("SENTBEFORE").SP().String(criteria.SentBefore.Format(searchDateLayout))
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

	for _, not := range criteria.Not {
		encodeItem("NOT").SP()
		writeSearchKey(enc, &not)
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
