package imapclient

type SortKey string

const (
	SortKeyArrival SortKey = "ARRIVAL"
	SortKeyCc      SortKey = "CC"
	SortKeyDate    SortKey = "DATE"
	SortKeyFrom    SortKey = "FROM"
	SortKeySize    SortKey = "SIZE"
	SortKeySubject SortKey = "SUBJECT"
	SortKeyTo      SortKey = "TO"
)

type SortCriterion struct {
	Key     SortKey
	Reverse bool
}

// SortOptions contains options for the SORT command.
type SortOptions struct {
	SearchCriteria *SearchCriteria
	SortCriteria   []SortCriterion
}

func (c *Client) sort(uid bool, options *SortOptions) *SortCommand {
	cmd := &SortCommand{}
	enc := c.beginCommand(uidCmdName("SORT", uid), cmd)
	enc.SP().List(len(options.SortCriteria), func(i int) {
		criterion := options.SortCriteria[i]
		if criterion.Reverse {
			enc.Atom("REVERSE").SP()
		}
		enc.Atom(string(criterion.Key))
	})
	enc.SP().Atom("UTF-8").SP()
	writeSearchKey(enc.Encoder, options.SearchCriteria)
	enc.end()
	return cmd
}

func (c *Client) handleSort() error {
	cmd := findPendingCmdByType[*SortCommand](c)
	for c.dec.SP() {
		var num uint32
		if !c.dec.ExpectNumber(&num) {
			return c.dec.Err()
		}
		if cmd != nil {
			cmd.nums = append(cmd.nums, num)
		}
	}
	return nil
}

// Sort sends a SORT command.
//
// This command requires support for the SORT extension.
func (c *Client) Sort(options *SortOptions) *SortCommand {
	return c.sort(false, options)
}

// UIDSort sends a UID SORT command.
//
// See Sort.
func (c *Client) UIDSort(options *SortOptions) *SortCommand {
	return c.sort(true, options)
}

// SortCommand is a SORT command.
type SortCommand struct {
	cmd
	nums []uint32
}

func (cmd *SortCommand) Wait() ([]uint32, error) {
	err := cmd.cmd.Wait()
	return cmd.nums, err
}
