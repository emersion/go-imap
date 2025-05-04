package imapmemserver

import (
	"sort"
	"strings"
	"sync"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

type Namespace struct {
	prefix string

	mutex           sync.Mutex
	mailboxes       map[string]*Mailbox
	prevUidValidity uint32
}

func NewNamespace(prefix string) *Namespace {
	return &Namespace{
		prefix:    prefix,
		mailboxes: make(map[string]*Mailbox),
	}
}

func (ns *Namespace) mailboxLocked(name string) (*Mailbox, error) {
	mbox := ns.mailboxes[name]
	if mbox == nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeNonExistent,
			Text: "No such mailbox",
		}
	}
	return mbox, nil
}

func (ns *Namespace) mailbox(name string) (*Mailbox, error) {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()
	return ns.mailboxLocked(name)
}

func (ns *Namespace) Status(name string, options *imap.StatusOptions) (*imap.StatusData, error) {
	mbox, err := ns.mailbox(name)
	if err != nil {
		return nil, err
	}
	return mbox.StatusData(options), nil
}

func (ns *Namespace) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	// TODO: fail if ref doesn't exist

	if len(patterns) == 0 {
		return w.WriteList(&imap.ListData{
			Attrs: []imap.MailboxAttr{imap.MailboxAttrNoSelect},
			Delim: mailboxDelim,
		})
	}

	var l []imap.ListData
	for name, mbox := range ns.mailboxes {
		match := false
		for _, pattern := range patterns {
			match = imapserver.MatchList(name, mailboxDelim, ref, pattern)
			if match {
				break
			}
		}
		if !match {
			continue
		}

		data := mbox.list(options)
		if data != nil {
			l = append(l, *data)
		}
	}

	sort.Slice(l, func(i, j int) bool {
		return l[i].Mailbox < l[j].Mailbox
	})

	for _, data := range l {
		if err := w.WriteList(&data); err != nil {
			return err
		}
	}

	return nil
}

func (ns *Namespace) Append(mailbox string, r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	mbox, err := ns.mailbox(mailbox)
	if err != nil {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeTryCreate,
			Text: "No such mailbox",
		}
	}
	return mbox.appendLiteral(r, options)
}

func (ns *Namespace) Create(name string, options *imap.CreateOptions) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	name = strings.TrimRight(name, string(mailboxDelim))

	if ns.mailboxes[name] != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAlreadyExists,
			Text: "Mailbox already exists",
		}
	}

	// UIDVALIDITY must change if a mailbox is deleted and re-created with the
	// same name.
	ns.prevUidValidity++
	ns.mailboxes[name] = NewMailbox(name, ns.prevUidValidity)
	return nil
}

func (ns *Namespace) Delete(name string) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if _, err := ns.mailboxLocked(name); err != nil {
		return err
	}

	delete(ns.mailboxes, name)
	return nil
}

func (ns *Namespace) Rename(oldName, newName string) error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	newName = strings.TrimRight(newName, string(mailboxDelim))

	mbox, err := ns.mailboxLocked(oldName)
	if err != nil {
		return err
	}

	if ns.mailboxes[newName] != nil {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeAlreadyExists,
			Text: "Mailbox already exists",
		}
	}

	mbox.rename(newName)
	ns.mailboxes[newName] = mbox
	delete(ns.mailboxes, oldName)
	return nil
}

func (ns *Namespace) Subscribe(name string) error {
	mbox, err := ns.mailbox(name)
	if err != nil {
		return err
	}
	mbox.SetSubscribed(true)
	return nil
}

func (ns *Namespace) Unsubscribe(name string) error {
	mbox, err := ns.mailbox(name)
	if err != nil {
		return err
	}
	mbox.SetSubscribed(false)
	return nil
}

func (ns *Namespace) NamespaceDescriptor() *imap.NamespaceDescriptor {
	return &imap.NamespaceDescriptor{
		Prefix: ns.prefix,
		Delim:  mailboxDelim,
	}
}
