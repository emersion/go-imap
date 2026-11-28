package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// Store sends a STORE command.
//
// Unless StoreFlags.Silent is set, the server will return the updated values.
//
// A nil options pointer is equivalent to a zero options value.
func (c *Client) Store(numSet imap.NumSet, store *imap.StoreFlags, options *imap.StoreOptions) *StoreCommand {
	cmd := &StoreCommand{
		FetchCommand: FetchCommand{
			numSet: numSet,
			msgs:   make(chan *FetchMessageData, 128),
		},
	}
	enc := c.beginCommand(uidCmdName("STORE", imapwire.NumSetKind(numSet)), cmd)
	enc.SP().NumSet(numSet).SP()
	if options != nil && options.UnchangedSince != 0 {
		enc.Special('(').Atom("UNCHANGEDSINCE").SP().ModSeq(options.UnchangedSince).Special(')').SP()
	}
	switch store.Op {
	case imap.StoreFlagsSet:
		// nothing to do
	case imap.StoreFlagsAdd:
		enc.Special('+')
	case imap.StoreFlagsDel:
		enc.Special('-')
	default:
		panic(fmt.Errorf("imapclient: unknown store flags op: %v", store.Op))
	}
	enc.Atom("FLAGS")
	if store.Silent {
		enc.Atom(".SILENT")
	}
	enc.SP().List(len(store.Flags), func(i int) {
		enc.Flag(store.Flags[i])
	})
	enc.end()
	return cmd
}

type StoreCommand struct {
	FetchCommand
	data *imap.StoreData
}

// Data returns additional information returned in tagged STORE
// response.
//
// It is nil until command completes (Collect, Close) and may be
// nil if command returned no additional information.
func (c *StoreCommand) Data() *imap.StoreData {
	return c.data
}
