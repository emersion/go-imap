package memory

import (
	"github.com/emersion/imap/common"
)

type Mailbox struct {
	name string
}

func (mbox *Mailbox) Info() (*common.MailboxInfo, error) {
	info := &common.MailboxInfo{
		Delimiter: "/",
		Name: mbox.name,
	}
	return info, nil
}
