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
			MailboxName:  mailbox,
			HostName:     hostname,
		}
	}

	return list, err
}

// FetchEnvelope returns a message's envelope from its header.
func FetchEnvelope(h message.Header) (*imap.Envelope, error) {
	mh := mail.Header{h}

	env := new(imap.Envelope)
	env.Date, _ = mh.Date()
	env.Subject, _ = mh.Subject()
	env.From, _ = headerAddressList(mh, "From")
	env.Sender, _ = headerAddressList(mh, "Sender")
	env.ReplyTo, _ = headerAddressList(mh, "Reply-To")
	env.To, _ = headerAddressList(mh, "To")
	env.Cc, _ = headerAddressList(mh, "Cc")
	env.Bcc, _ = headerAddressList(mh, "Bcc")
	env.InReplyTo = mh.Get("In-Reply-To")
	env.MessageId = mh.Get("Message-Id")

	return env, nil
}
