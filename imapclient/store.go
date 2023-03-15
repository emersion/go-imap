package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

func (c *Client) store(uid bool, seqSet imap.SeqSet, store *StoreFlags) *FetchCommand {
	cmd := &FetchCommand{msgs: make(chan *FetchMessageData, 128)}
	enc := c.beginCommand(uidCmdName("STORE", uid), cmd)
	enc.SP().Atom(seqSet.String()).SP()
	switch store.Op {
	case StoreFlagsSet:
		// nothing to do
	case StoreFlagsAdd:
		enc.Special('+')
	case StoreFlagsDel:
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
func (c *Client) Store(seqSet imap.SeqSet, store *StoreFlags) *FetchCommand {
	return c.store(false, seqSet, store)
}

// UIDStore sends a UID STORE command.
//
// See Store.
func (c *Client) UIDStore(seqSet imap.SeqSet, store *StoreFlags) *FetchCommand {
	return c.store(true, seqSet, store)
}

// StoreFlagsOp is a flag operation: set, add or delete.
type StoreFlagsOp int

const (
	StoreFlagsSet StoreFlagsOp = iota
	StoreFlagsAdd
	StoreFlagsDel
)

// StoreFlags alters message flags.
type StoreFlags struct {
	Op     StoreFlagsOp
	Silent bool
	Flags  []imap.Flag
}
