package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

func (c *Client) store(uid bool, seqSet imap.SeqSet, store *imap.StoreFlags) *FetchCommand {
	cmd := &FetchCommand{msgs: make(chan *FetchMessageData, 128)}
	enc := c.beginCommand(uidCmdName("STORE", uid), cmd)
	enc.SP().Atom(seqSet.String()).SP()
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
func (c *Client) Store(seqSet imap.SeqSet, store *imap.StoreFlags) *FetchCommand {
	return c.store(false, seqSet, store)
}

// UIDStore sends a UID STORE command.
//
// See Store.
func (c *Client) UIDStore(seqSet imap.SeqSet, store *imap.StoreFlags) *FetchCommand {
	return c.store(true, seqSet, store)
}
