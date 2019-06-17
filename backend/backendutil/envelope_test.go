package backendutil

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/textproto"
)

var testEnvelope = &imap.Envelope{
	Date:      testDate,
	Subject:   "Your Name.",
	From:      []*imap.Address{{PersonalName: "Mitsuha Miyamizu", MailboxName: "mitsuha.miyamizu", HostName: "example.org"}},
	Sender:    []*imap.Address{{PersonalName: "Mitsuha Miyamizu", MailboxName: "mitsuha.miyamizu", HostName: "example.org"}},
	ReplyTo:   []*imap.Address{{PersonalName: "Mitsuha Miyamizu", MailboxName: "mitsuha.miyamizu+replyto", HostName: "example.org"}},
	To:        []*imap.Address{{PersonalName: "Taki Tachibana", MailboxName: "taki.tachibana", HostName: "example.org"}},
	Cc:        []*imap.Address{},
	Bcc:       []*imap.Address{},
	InReplyTo: "",
	MessageId: "42@example.org",
}

func TestFetchEnvelope(t *testing.T) {
	hdr, err := textproto.ReadHeader(bufio.NewReader(strings.NewReader(testMailString)))
	if err != nil {
		t.Fatal("Expected no error while reading mail, got:", err)
	}

	env, err := FetchEnvelope(hdr)
	if err != nil {
		t.Fatal("Expected no error while fetching envelope, got:", err)
	}

	if !reflect.DeepEqual(env, testEnvelope) {
		t.Errorf("Expected envelope \n%+v\n but got \n%+v", testEnvelope, env)
	}
}

func TestFetchEnvelopeBase64(t *testing.T) {
	hdr, err := textproto.ReadHeader(bufio.NewReader(strings.NewReader(testBase64MailString)))
	if err != nil {
		t.Fatal("Expected no error while reading mail, got:", err)
	}

	env, err := FetchEnvelope(hdr)
	if err != nil {
		t.Fatal("Expected no error while fetching envelope, got:", err)
	}
	if len(env.From) != 1 {
		t.Fatal("Expected 1 address in envelope.From, got", len(env.From))
	}
	from, err := charset.DecodeHeader(env.From[0].PersonalName)
	if err != nil {
		t.Fatal("Expected no error while decoding PersonalName, got", err)
	}
	if from != testEnvelope.From[0].PersonalName {
		t.Errorf("Expected PersonalName \n%+v\n, but got \n%+v", testEnvelope.From[0].PersonalName, from)
	}
}
