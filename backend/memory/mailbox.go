package memory

import (
	"errors"
	"time"

	"github.com/emersion/go-imap/common"
)

type Mailbox struct {
	name string
	subscribed bool
	messages []*Message
	user *User
}

func (mbox *Mailbox) Name() string {
	return mbox.name
}

func (mbox *Mailbox) Info() (*common.MailboxInfo, error) {
	info := &common.MailboxInfo{
		Delimiter: "/",
		Name: mbox.name,
		Attributes: []string{"\\Noinferiors"},
	}
	return info, nil
}

func (mbox *Mailbox) uidNext() (uid uint32) {
	for _, msg := range mbox.messages {
		if msg.Uid > uid {
			uid = msg.Uid
		}
	}
	uid++
	return
}

func (mbox *Mailbox) Status(items []string) (*common.MailboxStatus, error) {
	status := &common.MailboxStatus{
		Items: items,
		Name: mbox.name,
		Flags: []string{"\\Answered", "\\Flagged", "\\Deleted", "\\Seen", "\\Draft"},
		PermanentFlags: []string{"\\Answered", "\\Flagged", "\\Deleted", "\\Seen", "\\Draft", "\\*"},
	}

	for _, name := range items {
		switch name {
		case "MESSAGES":
			status.Messages = uint32(len(mbox.messages))
		case "UIDNEXT":
			status.UidNext = mbox.uidNext()
		case "UIDVALIDITY":
			status.UidValidity = 1
		case "RECENT":
			status.Recent = 0
		case "UNSEEN":
			status.Unseen = 0
		}
	}

	return status, nil
}

func (mbox *Mailbox) Subscribe() error {
	mbox.subscribed = true
	return nil
}

func (mbox *Mailbox) Unsubscribe() error {
	mbox.subscribed = false
	return nil
}

func (mbox *Mailbox) Check() error {
	return nil
}

func (mbox *Mailbox) ListMessages(uid bool, seqset *common.SeqSet, items []string, ch chan<- *common.Message) (err error) {
	defer close(ch)

	for i, msg := range mbox.messages {
		seqid := uint32(i+1)

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = seqid
		}
		if !seqset.Contains(id) {
			continue
		}

		m := msg.Metadata(items)
		m.Id = seqid
		ch <- m
	}

	return
}

func (mbox *Mailbox) SearchMessages(uid bool, criteria *common.SearchCriteria) (ids []uint32, err error) {
	for i, msg := range mbox.messages {
		if !msg.Matches(criteria) {
			continue
		}

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i+1)
		}
		ids = append(ids, id)
	}

	return
}

func (mbox *Mailbox) CreateMessage(flags []string, date *time.Time, body []byte) error {
	if date == nil {
		now := time.Now()
		date = &now
	}

	mbox.messages = append(mbox.messages, &Message{&common.Message{
		Uid: mbox.uidNext(),
		Envelope: &common.Envelope{},
		BodyStructure: &common.BodyStructure{MimeType: "text", MimeSubType: "plain"},
		Size: uint32(len(body)),
		InternalDate: date,
		Flags: flags,
	}, body})

	return nil
}

func (mbox *Mailbox) UpdateMessagesFlags(uid bool, seqset *common.SeqSet, op common.FlagsOp, flags []string) error {
	for i, msg := range mbox.messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i+1)
		}
		if !seqset.Contains(id) {
			continue
		}

		switch op {
		case common.SetFlags:
			// TODO: keep \Recent if it is present
			msg.Flags = flags
		case common.AddFlags:
			// TODO: check for duplicates
			msg.Flags = append(msg.Flags, flags...)
		case common.RemoveFlags:
			// Iterate through flags from the last one to the first one, to be able to
			// delete some of them.
			for i := len(msg.Flags) - 1; i >= 0; i-- {
				flag := msg.Flags[i]

				for _, removeFlag := range flags {
					if removeFlag == flag {
						msg.Flags = append(msg.Flags[:i], msg.Flags[i+1:]...)
						break
					}
				}
			}
		}
	}

	return nil
}

func (mbox *Mailbox) CopyMessages(uid bool, seqset *common.SeqSet, destName string) error {
	dest, ok := mbox.user.mailboxes[destName]
	if !ok {
		return errors.New("Destination mailbox doesn't exist")
	}

	for i, msg := range mbox.messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i+1)
		}
		if !seqset.Contains(id) {
			continue
		}

		msgCopy := *msg
		msgCopy.Uid = dest.uidNext()
		dest.messages = append(dest.messages, &msgCopy)
	}

	return nil
}

func (mbox *Mailbox) Expunge() error {
	for i := len(mbox.messages) - 1; i >= 0; i-- {
		msg := mbox.messages[i]

		deleted := false
		for _, flag := range msg.Flags {
			if flag == "\\Deleted" {
				deleted = true
				break
			}
		}

		if deleted {
			mbox.messages = append(mbox.messages[:i], mbox.messages[i+1:]...)
		}
	}

	return nil
}
