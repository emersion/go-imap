package imapmemserver

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	gomessage "github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/emersion/go-message/textproto"
)

type message struct {
	// immutable
	uid imap.UID
	buf []byte
	t   time.Time

	// mutable, protected by Mailbox.mutex
	flags map[imap.Flag]struct{}
}

func (msg *message) fetch(w *imapserver.FetchResponseWriter, options *imap.FetchOptions) error {
	w.WriteUID(msg.uid)

	if options.Flags {
		w.WriteFlags(msg.flagList())
	}
	if options.InternalDate {
		w.WriteInternalDate(msg.t)
	}
	if options.RFC822Size {
		w.WriteRFC822Size(int64(len(msg.buf)))
	}
	if options.Envelope {
		w.WriteEnvelope(msg.envelope())
	}
	if options.BodyStructure != nil {
		w.WriteBodyStructure(imapserver.ExtractBodyStructure(bytes.NewReader(msg.buf)))
	}

	for _, bs := range options.BodySection {
		buf := imapserver.ExtractBodySection(bytes.NewReader(msg.buf), bs)
		wc := w.WriteBodySection(bs, int64(len(buf)))
		_, writeErr := wc.Write(buf)
		closeErr := wc.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}

	for _, bs := range options.BinarySection {
		buf := imapserver.ExtractBinarySection(bytes.NewReader(msg.buf), bs)
		wc := w.WriteBinarySection(bs, int64(len(buf)))
		_, writeErr := wc.Write(buf)
		closeErr := wc.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}

	for _, bss := range options.BinarySectionSize {
		n := imapserver.ExtractBinarySectionSize(bytes.NewReader(msg.buf), bss)
		w.WriteBinarySectionSize(bss, n)
	}

	return w.Close()
}

func (msg *message) envelope() *imap.Envelope {
	br := bufio.NewReader(bytes.NewReader(msg.buf))
	header, err := textproto.ReadHeader(br)
	if err != nil {
		return nil
	}
	return imapserver.ExtractEnvelope(header)
}

func (msg *message) flagList() []imap.Flag {
	var flags []imap.Flag
	for flag := range msg.flags {
		flags = append(flags, flag)
	}
	return flags
}

func (msg *message) store(store *imap.StoreFlags) {
	switch store.Op {
	case imap.StoreFlagsSet:
		msg.flags = make(map[imap.Flag]struct{})
		fallthrough
	case imap.StoreFlagsAdd:
		for _, flag := range store.Flags {
			msg.flags[canonicalFlag(flag)] = struct{}{}
		}
	case imap.StoreFlagsDel:
		for _, flag := range store.Flags {
			delete(msg.flags, canonicalFlag(flag))
		}
	default:
		panic(fmt.Errorf("unknown STORE flag operation: %v", store.Op))
	}
}

func (msg *message) search(seqNum uint32, criteria *imap.SearchCriteria) bool {
	for _, seqSet := range criteria.SeqNum {
		if seqNum == 0 || !seqSet.Contains(seqNum) {
			return false
		}
	}
	for _, uidSet := range criteria.UID {
		if !uidSet.Contains(msg.uid) {
			return false
		}
	}
	if !matchDate(msg.t, criteria.Since, criteria.Before) {
		return false
	}

	for _, flag := range criteria.Flag {
		if _, ok := msg.flags[canonicalFlag(flag)]; !ok {
			return false
		}
	}
	for _, flag := range criteria.NotFlag {
		if _, ok := msg.flags[canonicalFlag(flag)]; ok {
			return false
		}
	}

	if criteria.Larger != 0 && int64(len(msg.buf)) <= criteria.Larger {
		return false
	}
	if criteria.Smaller != 0 && int64(len(msg.buf)) >= criteria.Smaller {
		return false
	}

	if !matchBytes(msg.buf, criteria.Text) {
		return false
	}

	br := bufio.NewReader(bytes.NewReader(msg.buf))
	rawHeader, _ := textproto.ReadHeader(br)
	header := mail.Header{gomessage.Header{rawHeader}}

	for _, fieldCriteria := range criteria.Header {
		if !header.Has(fieldCriteria.Key) {
			return false
		}
		if fieldCriteria.Value == "" {
			continue
		}
		found := false
		for _, v := range header.Values(fieldCriteria.Key) {
			found = strings.Contains(strings.ToLower(v), strings.ToLower(fieldCriteria.Value))
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	if !criteria.SentSince.IsZero() || !criteria.SentBefore.IsZero() {
		t, err := header.Date()
		if err != nil {
			return false
		} else if !matchDate(t, criteria.SentSince, criteria.SentBefore) {
			return false
		}
	}

	if len(criteria.Body) > 0 {
		body, _ := io.ReadAll(br)
		if !matchBytes(body, criteria.Body) {
			return false
		}
	}

	for _, not := range criteria.Not {
		if msg.search(seqNum, &not) {
			return false
		}
	}
	for _, or := range criteria.Or {
		if !msg.search(seqNum, &or[0]) && !msg.search(seqNum, &or[1]) {
			return false
		}
	}

	return true
}

func matchDate(t, since, before time.Time) bool {
	// We discard time zone information by setting it to UTC.
	// RFC 3501 explicitly requires zone unaware date comparison.
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

	if !since.IsZero() && t.Before(since) {
		return false
	}
	if !before.IsZero() && !t.Before(before) {
		return false
	}
	return true
}

func matchBytes(buf []byte, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	buf = bytes.ToLower(buf)
	for _, s := range patterns {
		if !bytes.Contains(buf, bytes.ToLower([]byte(s))) {
			return false
		}
	}
	return true
}

func canonicalFlag(flag imap.Flag) imap.Flag {
	return imap.Flag(strings.ToLower(string(flag)))
}
