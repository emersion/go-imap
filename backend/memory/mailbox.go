package memory

import (
	"errors"
	"time"

	"github.com/emersion/go-imap"
)

type Mailbox struct {
	name       string
	subscribed bool
	messages   []*Message
	user       *User
}

func (mbox *Mailbox) Name() string {
	return mbox.name
}

func (mbox *Mailbox) Info() (*imap.MailboxInfo, error) {
	info := &imap.MailboxInfo{
		Delimiter:  "/",
		Name:       mbox.name,
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

func (mbox *Mailbox) Status(items []string) (*imap.MailboxStatus, error) {
	status := &imap.MailboxStatus{
		Items:          items,
		Name:           mbox.name,
		Flags:          []string{"\\Answered", "\\Flagged", "\\Deleted", "\\Seen", "\\Draft"},
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

func (mbox *Mailbox) ListMessages(uid bool, seqset *imap.SeqSet, items []string, ch chan<- *imap.Message) (err error) {
	defer close(ch)

	for i, msg := range mbox.messages {
		seqNum := uint32(i + 1)

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = seqNum
		}
		if !seqset.Contains(id) {
			continue
		}

		m := msg.Metadata(items)
		m.SeqNum = seqNum
		ch <- m
	}

	return
}

func (mbox *Mailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) (ids []uint32, err error) {
	for i, msg := range mbox.messages {
		if !msg.Matches(criteria) {
			continue
		}

		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i + 1)
		}
		ids = append(ids, id)
	}

	return
}

func (mbox *Mailbox) CreateMessage(flags []string, date time.Time, body []byte) error {
	if date.IsZero() {
		date = time.Now()
	}

	mbox.messages = append(mbox.messages, &Message{&imap.Message{
		Uid:           mbox.uidNext(),
		Envelope:      &imap.Envelope{},
		BodyStructure: &imap.BodyStructure{MimeType: "text", MimeSubType: "plain"},
		Size:          uint32(len(body)),
		InternalDate:  date,
		Flags:         flags,
	}, body})

	return nil
}

func (mbox *Mailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, op imap.FlagsOp, flags []string) error {
	for i, msg := range mbox.messages {
		var id uint32
		if uid {
			id = msg.Uid
		} else {
			id = uint32(i + 1)
		}
		if !seqset.Contains(id) {
			continue
		}

		switch op {
		case imap.SetFlags:
			// TODO: keep \Recent if it is present
			msg.Flags = flags
		case imap.AddFlags:
			// TODO: check for duplicates
			msg.Flags = append(msg.Flags, flags...)
		case imap.RemoveFlags:
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

func (mbox *Mailbox) CopyMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	dest, ok := mbox.user.mailboxes[destName]
	if !ok {
		return errors.New("Destination mailbox doesn't exist")
	}

	for i, msg := range mbox.messages {
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
