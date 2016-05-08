package memory

import (
	"github.com/emersion/imap/common"
)

type Mailbox struct {
	name string
	messages []*Message
}

func (mbox *Mailbox) Info() (*common.MailboxInfo, error) {
	info := &common.MailboxInfo{
		Delimiter: "/",
		Name: mbox.name,
	}
	return info, nil
}

func (mbox *Mailbox) uidNext() (uid uint32) {
	for _, msg := range mbox.messages {
		if msg.uid > uid {
			uid = msg.uid
		}
	}
	return
}

func (mbox *Mailbox) Status(items []string) (*common.MailboxStatus, error) {
	status := &common.MailboxStatus{
		Items: items,
		Name: mbox.name,
	}

	for _, name := range items {
		switch name {
		case "MESSAGES":
			status.Messages = uint32(len(mbox.messages))
		case "UIDNEXT":
			status.UidNext = mbox.uidNext()
		}
	}

	return status, nil
}
