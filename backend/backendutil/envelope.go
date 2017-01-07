package backendutil

import (
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

func headerAddressList(h mail.Header, key string) ([]*imap.Address, error) {
	addrs, err := h.AddressList(key)

	list := make([]*imap.Address, len(addrs))
	for i, a := range addrs {
		parts := strings.SplitN(a.Address, "@", 2)
		mailbox := parts[0]
		var hostname string
		if len(parts) == 2 {
			hostname = parts[1]
		}

		list[i] = &imap.Address{
			PersonalName: a.Name,
			MailboxName: mailbox,
			HostName: hostname,
		}
	}

	return list, err
}

func FetchEnvelope(e *message.Entity) (*imap.Envelope, error) {
	h := mail.Header{e.Header}

	env := new(imap.Envelope)
	env.Date, _ = h.Date()
	env.Subject, _ = h.Subject()
	env.From, _ = headerAddressList(h, "From")
	env.Sender, _ = headerAddressList(h, "Sender")
	env.ReplyTo, _ = headerAddressList(h, "Reply-To")
	env.To, _ = headerAddressList(h, "To")
	env.Cc, _ = headerAddressList(h, "Cc")
	env.Bcc, _ = headerAddressList(h, "Bcc")
	env.InReplyTo = h.Get("In-Reply-To")
	env.MessageId = h.Get("Message-Id")

	return env, nil
}
