package memory

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/backendutil"
)

var Delimiter = "/"

type Mailbox struct {
	sync.RWMutex

	Flags       []string
	Attributes  []string
	Subscribed  bool
	Messages    []*Message
	UidValidity uint32

	name string
	user *User
}

func NewMailbox(user *User, name string, specialUse string) *Mailbox {
	mbox := &Mailbox{
		name: name, user: user,
		UidValidity: 1, // Use 1 for tests.  Should use timestamp instead.
		Messages:    []*Message{},
		Flags: []string{
			imap.AnsweredFlag,
			imap.FlaggedFlag,
			imap.DeletedFlag,
			imap.SeenFlag,
			imap.DraftFlag,
			"nonjunk",
		},
	}
	if specialUse != "" {
		mbox.Attributes = []string{specialUse}
	}
	return mbox
}

func (mbox *Mailbox) Name() string {
	return mbox.name
}

func (mbox *Mailbox) Info() (*imap.MailboxInfo, error) {
	mbox.RLock()
	defer mbox.RUnlock()

	info := &imap.MailboxInfo{
		Attributes: mbox.Attributes,
		Delimiter:  Delimiter,
		Name:       mbox.name,
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

func (mbox *Mailbox) status(items []imap.StatusItem, flags bool) (*imap.MailboxStatus, error) {
	status := imap.NewMailboxStatus(mbox.name, items)
	if flags {
		// Copy flags slice (don't re-use slice)
		flags := append(mbox.Flags[:0:0], mbox.Flags...)
		status.Flags = flags
		status.PermanentFlags = append(flags, "\\*")
	}
	status.UnseenSeqNum = mbox.unseenSeqNum()

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = uint32(len(mbox.Messages))
		case imap.StatusUidNext:
			status.UidNext = mbox.uidNext()
		case imap.StatusUidValidity:
			status.UidValidity = mbox.UidValidity
		case imap.StatusRecent:
			status.Recent = 0 // TODO
		case imap.StatusUnseen:
			status.Unseen = 0 // TODO
		}
	}

	return status, nil
}

func (mbox *Mailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	mbox.RLock()
	defer mbox.RUnlock()

	return mbox.status(items, true)
}

func (mbox *Mailbox) SetSubscribed(subscribed bool) error {
	mbox.Lock()
	defer mbox.Unlock()

	mbox.Subscribed = subscribed

	return nil
}

func (mbox *Mailbox) Check() error {
	return nil
}

func (mbox *Mailbox) ListMessages(uid bool, seqSet *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	mbox.RLock()
	defer mbox.RUnlock()
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

	return nil
}

func (mbox *Mailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	mbox.RLock()
	defer mbox.RUnlock()

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
	return ids, nil
}

func (mbox *Mailbox) CreateMessage(flags []string, date time.Time, body imap.Literal) error {
	mbox.Lock()
	defer mbox.Unlock()

	if date.IsZero() {
		date = time.Now()
	}

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	mbox.Messages = append(mbox.Messages, &Message{
		Uid:   mbox.uidNext(),
		Date:  date,
		Size:  uint32(len(b)),
		Flags: append(flags, imap.RecentFlag),
		Body:  b,
	})
	mbox.Flags = backendutil.UpdateFlags(mbox.Flags, imap.AddFlags, flags)
	mbox.user.PushMailboxUpdate(mbox)
	return nil
}

func (mbox *Mailbox) pushMessageUpdate(msg *Message, seqNum uint32) {
	uMsg := imap.NewMessage(seqNum, []imap.FetchItem{imap.FetchFlags})
	uMsg.Flags = msg.Flags
	mbox.user.PushMessageUpdate(mbox.name, uMsg)
}

func CompareFlags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func UpdateFlags(current []string, op imap.FlagsOp, flags []string) ([]string, bool) {
	origFlags := append(current[:0:0], current...)
	current = backendutil.UpdateFlags(current, op, flags)
	changed := !CompareFlags(current, origFlags)
	return current, changed
}

func (mbox *Mailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, op imap.FlagsOp, flags []string) error {
	mbox.Lock()
	defer mbox.Unlock()

	// Update mailbox flags list
	if op == imap.AddFlags || op == imap.SetFlags {
		if newFlags, changed := UpdateFlags(mbox.Flags, imap.AddFlags, flags); changed {
			mbox.Flags = newFlags
			mbox.user.PushMailboxUpdate(mbox)
		}
	}

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

		if newFlags, changed := UpdateFlags(msg.Flags, op, flags); changed {
			msg.Flags = newFlags
			mbox.pushMessageUpdate(msg, uint32(i+1))
		}
	}

	return nil
}

func (mbox *Mailbox) CopyMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	mbox.Lock()
	defer mbox.Unlock()

	dest, ok := mbox.user.mailboxes[destName]
	if !ok {
		return backend.ErrNoSuchMailbox
	}

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

		msgCopy := *msg
		msgCopy.Uid = dest.uidNext()
		dest.Messages = append(dest.Messages, &msgCopy)
	}
	mbox.user.PushMailboxUpdate(dest)

	return nil
}

func (mbox *Mailbox) MoveMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	mbox.Lock()
	defer mbox.Unlock()

	dest, ok := mbox.user.mailboxes[destName]
	if !ok {
		return backend.ErrNoSuchMailbox
	}

	flags := []string{imap.DeletedFlag}
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

		msgCopy := *msg
		msgCopy.Uid = dest.uidNext()
		dest.Messages = append(dest.Messages, &msgCopy)
		// Mark source message as deleted
		msg.Flags = backendutil.UpdateFlags(msg.Flags, imap.AddFlags, flags)
	}

	mbox.user.PushMailboxUpdate(dest)
	mbox.user.PushMailboxUpdate(mbox)
	return mbox.expunge()
}

func (mbox *Mailbox) expunge() error {
	for i := len(mbox.Messages) - 1; i >= 0; i-- {
		msg := mbox.Messages[i]

		deleted := false
		for _, flag := range msg.Flags {
			if flag == imap.DeletedFlag {
				deleted = true
				break
			}
		}

		if deleted {
			mbox.Messages = append(mbox.Messages[:i], mbox.Messages[i+1:]...)
			// send expunge update
			mbox.user.PushExpungeUpdate(mbox.name, uint32(i+1))
		}
	}

	return nil
}

func (mbox *Mailbox) Expunge() error {
	mbox.Lock()
	defer mbox.Unlock()

	return mbox.expunge()
}
