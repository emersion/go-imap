package memory

import (
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/backendutil"
)

var Delimiter = "/"

type Mailbox struct {
	Subscribed bool
	Messages   []*Message

	name string
	user *User
}

type SelectedMailbox struct {
	*Mailbox
	conn     backend.Conn
	readOnly bool
}

func (mbox *Mailbox) Name() string {
	return mbox.name
}

func (mbox *Mailbox) info() (*imap.MailboxInfo, error) {
	info := &imap.MailboxInfo{
		Delimiter: Delimiter,
		Name:      mbox.name,
	}
	return info, nil
}

func (mbox *Mailbox) uidNext() uint32 {
	var uid uint32
	for _, msg := range mbox.Messages {
		if msg.Uid > uid {
			uid = msg.Uid
		}
	}
	uid++
	return uid
}

func (mbox *Mailbox) flags() []string {
	flagsMap := make(map[string]bool)
	for _, msg := range mbox.Messages {
		for _, f := range msg.Flags {
			if !flagsMap[f] {
				flagsMap[f] = true
			}
		}
	}

	var flags []string
	for f := range flagsMap {
		flags = append(flags, f)
	}
	return flags
}

func (mbox *Mailbox) unseenSeqNum() uint32 {
	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		seen := false
		for _, flag := range msg.Flags {
			if flag == imap.SeenFlag {
				seen = true
				break
			}
		}

		if !seen {
			return seqNum
		}
	}
	return 0
}

func (mbox *Mailbox) Poll(_ bool) error {
	return nil
}

func (mbox *Mailbox) ListMessages(uid bool, seqSet *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message, _ []backend.ExtensionOption) ([]backend.ExtensionResult, error) {
	defer close(ch)

	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = seqNum
		}
		if !seqSet.Contains(id) {
			continue
		}

		m, err := msg.Fetch(seqNum, items)
		if err != nil {
			continue
		}

		ch <- m
	}

	return nil, nil
}

func (mbox *Mailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria, _ []backend.ExtensionOption) ([]backend.ExtensionResult, []uint32, error) {
	var ids []uint32
	for i, msg := range mbox.Messages {
		seqNum := uint32(i + 1)

		ok, err := msg.Match(seqNum, criteria)
		if err != nil || !ok {
			continue
		}

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = seqNum
		}
		ids = append(ids, id)
	}
	return nil, ids, nil
}

func (mbox *SelectedMailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, op imap.FlagsOp,
	silent bool, flags []string, _ []backend.ExtensionOption) ([]backend.ExtensionResult, error) {
	for i, msg := range mbox.Messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		msg.Flags = backendutil.UpdateFlags(msg.Flags, op, flags)

		if !silent {
			updMsg := imap.NewMessage(uint32(i+1), []imap.FetchItem{imap.FetchFlags})
			updMsg.Flags = msg.Flags
			if uid {
				updMsg.Items[imap.FetchUid] = nil
				updMsg.Uid = msg.Uid
			}
			mbox.conn.SendUpdate(&backend.MessageUpdate{Message: updMsg})
		}
	}

	return nil, nil
}

func (mbox *Mailbox) CopyMessages(uid bool, seqset *imap.SeqSet, destName string, _ []backend.ExtensionOption) ([]backend.ExtensionResult, error) {
	dest, ok := mbox.user.mailboxes[destName]
	if !ok {
		return nil, backend.ErrNoSuchMailbox
	}

	var srcSet, destSet imap.SeqSet

	for i, msg := range mbox.Messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		srcSet.AddNum(msg.Uid)

		msgCopy := *msg
		msgCopy.Uid = dest.uidNext()
		dest.Messages = append(dest.Messages, &msgCopy)
		destSet.AddNum(msgCopy.Uid)
	}

	return []backend.ExtensionResult{
		backend.CopyUIDs{
			Source:      &srcSet,
			UIDValidity: 1,
			Dest:        &destSet,
		},
	}, nil
}

func (mbox *SelectedMailbox) Expunge(opts []backend.ExtensionOption) ([]backend.ExtensionResult, error) {
	var allowedUIDs *imap.SeqSet
	for _, opt := range opts {
		switch opt := opt.(type) {
		case backend.ExpungeSeqSet:
			allowedUIDs = opt.SeqSet
		}
	}

	for i := len(mbox.Messages) - 1; i >= 0; i-- {
		msg := mbox.Messages[i]

		deleted := false
		for _, flag := range msg.Flags {
			if flag == imap.DeletedFlag {
				deleted = true
				break
			}
		}

		if allowedUIDs != nil && !allowedUIDs.Contains(msg.Uid) {
			continue
		}

		if deleted {
			mbox.Messages = append(mbox.Messages[:i], mbox.Messages[i+1:]...)

			mbox.conn.SendUpdate(&backend.ExpungeUpdate{SeqNum: uint32(i + 1)})
		}
	}

	return nil, nil
}

func (mbox *Mailbox) Close() error {
	return nil
}
