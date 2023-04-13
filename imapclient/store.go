package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

func (c *Client) store(uid bool, seqSet imap.SeqSet, store *imap.StoreFlags, options *imap.StoreOptions) *FetchCommand {
	cmd := &FetchCommand{msgs: make(chan *FetchMessageData, 128)}
	enc := c.beginCommand(uidCmdName("STORE", uid), cmd)
	enc.SP().SeqSet(seqSet).SP()
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

// Store sends a STORE command.
//
// Unless StoreFlags.Silent is set, the server will return the updated values.
//
// A nil options pointer is equivalent to a zero options value.
func (c *Client) Store(seqSet imap.SeqSet, store *imap.StoreFlags, options *imap.StoreOptions) *FetchCommand {
	return c.store(false, seqSet, store, options)
}

// UIDStore sends a UID STORE command.
//
// See Store.
func (c *Client) UIDStore(seqSet imap.SeqSet, store *imap.StoreFlags, options *imap.StoreOptions) *FetchCommand {
	return c.store(true, seqSet, store, options)
}
