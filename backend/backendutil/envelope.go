package backendutil

import (
	"mime"
	"net/mail"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/textproto"
)

func headerAddressList(parser *mail.AddressParser, value string) ([]*imap.Address, error) {
	addrs, err := parser.ParseList(value)
	if err != nil {
		return []*imap.Address{}, err
	}

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
func FetchEnvelope(h textproto.Header) (*imap.Envelope, error) {
	env := new(imap.Envelope)

	parser := mail.AddressParser{&mime.WordDecoder{imap.CharsetReader}}
	env.Date, _ = mail.ParseDate(h.Get("Date"))
	env.Subject = h.Get("Subject")
	env.From, _ = headerAddressList(&parser, h.Get("From"))
	env.Sender, _ = headerAddressList(&parser, h.Get("Sender"))
	if len(env.Sender) == 0 {
		env.Sender = env.From
	}
	env.ReplyTo, _ = headerAddressList(&parser, h.Get("Reply-To"))
	if len(env.ReplyTo) == 0 {
		env.ReplyTo = env.From
	}
	env.To, _ = headerAddressList(&parser, h.Get("To"))
	env.Cc, _ = headerAddressList(&parser, h.Get("Cc"))
	env.Bcc, _ = headerAddressList(&parser, h.Get("Bcc"))
	env.InReplyTo = h.Get("In-Reply-To")
	env.MessageId = h.Get("Message-Id")

	return env, nil
}
