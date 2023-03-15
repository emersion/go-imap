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
	enc.Atom("ALL") // TODO: remove

	if len(criteria.SeqNum) > 0 {
		enc.SP().Atom(criteria.SeqNum.String())
	}
	if len(criteria.UID) > 0 {
		enc.SP().Atom("UID").SP().Atom(criteria.UID.String())
	}

	if !criteria.Since.IsZero() && !criteria.Before.IsZero() && criteria.Before.Sub(criteria.Since) == 24*time.Hour {
		enc.SP().Atom("ON").SP().String(criteria.Since.Format(searchDateLayout))
	} else {
		if !criteria.Since.IsZero() {
			enc.SP().Atom("SINCE").SP().String(criteria.Since.Format(searchDateLayout))
		}
		if !criteria.Before.IsZero() {
			enc.SP().Atom("BEFORE").SP().String(criteria.Before.Format(searchDateLayout))
		}
	}
	if !criteria.SentSince.IsZero() && !criteria.SentBefore.IsZero() && criteria.SentBefore.Sub(criteria.SentSince) == 24*time.Hour {
		enc.SP().Atom("SENTON").SP().String(criteria.SentSince.Format(searchDateLayout))
	} else {
		if !criteria.SentSince.IsZero() {
			enc.SP().Atom("SENTSINCE").SP().String(criteria.SentSince.Format(searchDateLayout))
		}
		if !criteria.SentBefore.IsZero() {
			enc.SP().Atom("SENTBEFORE").SP().String(criteria.SentBefore.Format(searchDateLayout))
		}
	}

	for _, kv := range criteria.Header {
		switch k := strings.ToUpper(kv.Key); k {
		case "BCC", "CC", "FROM", "SUBJECT", "TO":
			enc.SP().Atom(k)
		default:
			enc.SP().Atom("HEADER").SP().String(kv.Key)
		}
		enc.SP().String(kv.Value)
	}

	for _, s := range criteria.Body {
		enc.SP().Atom("BODY").SP().String(s)
	}
	for _, s := range criteria.Text {
		enc.SP().Atom("TEXT").SP().String(s)
	}

	for _, flag := range criteria.Flag {
		if k := flagSearchKey(flag); k != "" {
			enc.SP().Atom(k)
		} else {
			enc.SP().Atom("KEYWORD").SP().Flag(flag)
		}
	}
	for _, flag := range criteria.NotFlag {
		if k := flagSearchKey(flag); k != "" {
			enc.SP().Atom("UN" + k)
		} else {
			enc.SP().Atom("UNKEYWORD").SP().Flag(flag)
		}
	}

	if criteria.Larger > 0 {
		enc.SP().Atom("LARGER").SP().Number64(criteria.Larger)
	}
	if criteria.Smaller > 0 {
		enc.SP().Atom("SMALLER").SP().Number64(criteria.Smaller)
	}

	for _, not := range criteria.Not {
		enc.SP().Atom("NOT").SP()
		writeSearchKey(enc, &not)
	}
	for _, or := range criteria.Or {
		enc.SP().Atom("OR").SP()
		writeSearchKey(enc, &or[0])
		enc.SP()
		writeSearchKey(enc, &or[1])
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
