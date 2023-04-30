package imapmemserver

import (
	"bytes"
	"sort"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

// Mailbox is an in-memory mailbox.
//
// The same mailbox can be shared between multiple connections and multiple
// users.
type Mailbox struct {
	tracker     *imapserver.MailboxTracker
	uidValidity uint32

	mutex      sync.Mutex
	name       string
	subscribed bool
	l          []*message
	uidNext    uint32
}

// NewMailbox creates a new mailbox.
func NewMailbox(name string, uidValidity uint32) *Mailbox {
	return &Mailbox{
		tracker:     imapserver.NewMailboxTracker(0),
		uidValidity: uidValidity,
		name:        name,
		uidNext:     1,
	}
}

func (mbox *Mailbox) list(options *imap.ListOptions) *imap.ListData {
	mbox.mutex.Lock()
	defer mbox.mutex.Unlock()

	if options.SelectSubscribed && !mbox.subscribed {
		return nil
	}

	data := imap.ListData{
		Mailbox: mbox.name,
		Delim:   mailboxDelim,
	}
	if mbox.subscribed {
		data.Attrs = append(data.Attrs, imap.MailboxAttrSubscribed)
	}
	if options.ReturnStatus != nil {
		data.Status = mbox.statusDataLocked(options.ReturnStatus)
	}
	return &data
}

// StatusData returns data for the STATUS command.
func (mbox *Mailbox) StatusData(options *imap.StatusOptions) *imap.StatusData {
	mbox.mutex.Lock()
	defer mbox.mutex.Unlock()
	return mbox.statusDataLocked(options)
}

func (mbox *Mailbox) statusDataLocked(options *imap.StatusOptions) *imap.StatusData {
	data := imap.StatusData{Mailbox: mbox.name}
	if options.NumMessages {
		num := uint32(len(mbox.l))
		data.NumMessages = &num
	}
	if options.UIDNext {
		data.UIDNext = mbox.uidNext
	}
	if options.UIDValidity {
		data.UIDValidity = mbox.uidValidity
	}
	if options.NumUnseen {
		num := uint32(len(mbox.l)) - mbox.countByFlagLocked(imap.FlagSeen)
		data.NumUnseen = &num
	}
	if options.NumDeleted {
		num := mbox.countByFlagLocked(imap.FlagDeleted)
		data.NumUnseen = &num
	}
	if options.Size {
		size := mbox.sizeLocked()
		data.Size = &size
	}
	return &data
}

func (mbox *Mailbox) countByFlagLocked(flag imap.Flag) uint32 {
	var n uint32
	for _, msg := range mbox.l {
		if _, ok := msg.flags[canonicalFlag(flag)]; ok {
			n++
		}
	}
	return n
}

func (mbox *Mailbox) sizeLocked() int64 {
	var size int64
	for _, msg := range mbox.l {
		size += int64(len(msg.buf))
	}
	return size
}

func (mbox *Mailbox) appendLiteral(r imap.LiteralReader, options *imap.AppendOptions) (*imap.AppendData, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}
	return mbox.appendBytes(buf.Bytes(), options), nil
}

func (mbox *Mailbox) copyMsg(msg *message) *imap.AppendData {
	return mbox.appendBytes(msg.buf, &imap.AppendOptions{
		Time:  msg.t,
		Flags: msg.flagList(),
	})
}

func (mbox *Mailbox) appendBytes(buf []byte, options *imap.AppendOptions) *imap.AppendData {
	msg := &message{
		flags: make(map[imap.Flag]struct{}),
		buf:   buf,
	}

	if options.Time.IsZero() {
		msg.t = time.Now()
	} else {
		msg.t = options.Time
	}

	for _, flag := range options.Flags {
		msg.flags[canonicalFlag(flag)] = struct{}{}
	}

	mbox.mutex.Lock()
	defer mbox.mutex.Unlock()

	msg.uid = mbox.uidNext
	mbox.uidNext++

	mbox.l = append(mbox.l, msg)
	mbox.tracker.QueueNumMessages(uint32(len(mbox.l)))

	return &imap.AppendData{
		UIDValidity: mbox.uidValidity,
		UID:         msg.uid,
	}
}

func (mbox *Mailbox) rename(newName string) {
	mbox.mutex.Lock()
	mbox.name = newName
	mbox.mutex.Unlock()
}

// SetSubscribed changes the subscription state of this mailbox.
func (mbox *Mailbox) SetSubscribed(subscribed bool) {
	mbox.mutex.Lock()
	mbox.subscribed = subscribed
	mbox.mutex.Unlock()
}

func (mbox *Mailbox) selectDataLocked() *imap.SelectData {
	flags := mbox.flagsLocked()

	permanentFlags := make([]imap.Flag, len(flags))
	copy(permanentFlags, flags)
	permanentFlags = append(permanentFlags, imap.FlagWildcard)

	return &imap.SelectData{
		Flags:          flags,
		PermanentFlags: permanentFlags,
		NumMessages:    uint32(len(mbox.l)),
		UIDNext:        mbox.uidNext,
		UIDValidity:    mbox.uidValidity,
	}
}

func (mbox *Mailbox) flagsLocked() []imap.Flag {
	m := make(map[imap.Flag]struct{})
	for _, msg := range mbox.l {
		for flag := range msg.flags {
			m[flag] = struct{}{}
		}
	}

	var l []imap.Flag
	for flag := range m {
		l = append(l, flag)
	}

	sort.Slice(l, func(i, j int) bool {
		return l[i] < l[j]
	})

	return l
}

func (mbox *Mailbox) Expunge(w *imapserver.ExpungeWriter, uids *imap.SeqSet) error {
	expunged := make(map[*message]struct{})
	mbox.mutex.Lock()
	for _, msg := range mbox.l {
		if uids != nil && !uids.Contains(msg.uid) {
			continue
		}
		if _, ok := msg.flags[canonicalFlag(imap.FlagDeleted)]; ok {
			expunged[msg] = struct{}{}
		}
	}
	mbox.mutex.Unlock()

	if len(expunged) == 0 {
		return nil
	}

	mbox.mutex.Lock()
	mbox.expungeLocked(expunged)
	mbox.mutex.Unlock()

	return nil
}

func (mbox *Mailbox) expungeLocked(expunged map[*message]struct{}) (seqNums []uint32) {
	// TODO: optimize

	// Iterate in reverse order, to keep sequence numbers consistent
	var filtered []*message
	for i := len(mbox.l) - 1; i >= 0; i-- {
		msg := mbox.l[i]
		if _, ok := expunged[msg]; ok {
			seqNum := uint32(i) + 1
			seqNums = append(seqNums, seqNum)
			mbox.tracker.QueueExpunge(seqNum)
		} else {
			filtered = append(filtered, msg)
		}
	}

	// Reverse filtered
	for i := 0; i < len(filtered)/2; i++ {
		j := len(filtered) - i - 1
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	mbox.l = filtered

	return seqNums
}

// NewView creates a new view into this mailbox.
//
// Callers must call MailboxView.Close once they are done with the mailbox view.
func (mbox *Mailbox) NewView() *MailboxView {
	return &MailboxView{
		Mailbox: mbox,
		tracker: mbox.tracker.NewSession(),
	}
}

// A MailboxView is a view into a mailbox.
//
// Each view has its own queue of pending unilateral updates.
//
// Once the mailbox view is no longer used, Close must be called.
//
// Typically, a new MailboxView is created for each IMAP connection in the
// selected state.
type MailboxView struct {
	*Mailbox
	tracker *imapserver.SessionTracker
}

// Close releases the resources allocated for the mailbox view.
func (mbox *MailboxView) Close() {
	mbox.tracker.Close()
}

func (mbox *MailboxView) Fetch(w *imapserver.FetchWriter, numKind imapserver.NumKind, seqSet imap.SeqSet, items []imap.FetchItem, options *imap.FetchOptions) error {
	markSeen := false
	for _, item := range items {
		if item, ok := item.(*imap.FetchItemBodySection); ok && !item.Peek {
			markSeen = true
			break
		}
	}

	var err error
	mbox.forEach(numKind, seqSet, func(seqNum uint32, msg *message) {
		if err != nil {
			return
		}

		if markSeen {
			msg.flags[canonicalFlag(imap.FlagSeen)] = struct{}{}
			mbox.Mailbox.tracker.QueueMessageFlags(seqNum, msg.uid, msg.flagList(), nil)
		}

		respWriter := w.CreateMessage(mbox.tracker.EncodeSeqNum(seqNum))
		err = msg.fetch(respWriter, items)
	})
	return err
}

func (mbox *MailboxView) Search(numKind imapserver.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) (*imap.SearchData, error) {
	mbox.mutex.Lock()
	defer mbox.mutex.Unlock()

	for _, seqSet := range criteria.SeqNum {
		mbox.staticSeqSet(seqSet, imapserver.NumKindSeq)
	}
	for _, seqSet := range criteria.UID {
		mbox.staticSeqSet(seqSet, imapserver.NumKindUID)
	}

	data := imap.SearchData{
		UID: numKind == imapserver.NumKindUID,
	}

	for i, msg := range mbox.l {
		seqNum := mbox.tracker.EncodeSeqNum(uint32(i) + 1)

		if !msg.search(seqNum, criteria) {
			continue
		}

		var num uint32
		switch numKind {
		case imapserver.NumKindSeq:
			num = seqNum
		case imapserver.NumKindUID:
			num = msg.uid
		}
		if num == 0 {
			continue
		}
		data.All.AddNum(num)
		if data.Min == 0 || num < data.Min {
			data.Min = num
		}
		if data.Max == 0 || num > data.Max {
			data.Max = num
		}
		data.Count++
	}

	return &data, nil
}

func (mbox *MailboxView) Store(w *imapserver.FetchWriter, numKind imapserver.NumKind, seqSet imap.SeqSet, flags *imap.StoreFlags, options *imap.StoreOptions) error {
	mbox.forEach(numKind, seqSet, func(seqNum uint32, msg *message) {
		msg.store(flags)
		mbox.Mailbox.tracker.QueueMessageFlags(seqNum, msg.uid, msg.flagList(), mbox.tracker)
	})
	if !flags.Silent {
		return mbox.Fetch(w, numKind, seqSet, []imap.FetchItem{imap.FetchItemFlags}, nil)
	}
	return nil
}

func (mbox *MailboxView) Poll(w *imapserver.UpdateWriter, allowExpunge bool) error {
	return mbox.tracker.Poll(w, allowExpunge)
}

func (mbox *MailboxView) Idle(w *imapserver.UpdateWriter, stop <-chan struct{}) error {
	return mbox.tracker.Idle(w, stop)
}

func (mbox *MailboxView) forEach(numKind imapserver.NumKind, seqSet imap.SeqSet, f func(seqNum uint32, msg *message)) {
	mbox.mutex.Lock()
	defer mbox.mutex.Unlock()
	mbox.forEachLocked(numKind, seqSet, f)
}

func (mbox *MailboxView) forEachLocked(numKind imapserver.NumKind, seqSet imap.SeqSet, f func(seqNum uint32, msg *message)) {
	// TODO: optimize

	mbox.staticSeqSet(seqSet, numKind)

	for i, msg := range mbox.l {
		seqNum := uint32(i) + 1

		var num uint32
		switch numKind {
		case imapserver.NumKindSeq:
			num = mbox.tracker.EncodeSeqNum(seqNum)
		case imapserver.NumKindUID:
			num = msg.uid
		}
		if num == 0 || !seqSet.Contains(num) {
			continue
		}

		f(seqNum, msg)
	}
}

// staticSeqSet converts a dynamic sequence set into a static one.
//
// This is necessary to properly handle the special symbol "*", which
// represents the maximum sequence number or UID in the mailbox.
func (mbox *MailboxView) staticSeqSet(seqSet imap.SeqSet, numKind imapserver.NumKind) {
	var max uint32
	switch numKind {
	case imapserver.NumKindSeq:
		max = uint32(len(mbox.l))
	case imapserver.NumKindUID:
		max = mbox.uidNext - 1
	}

	for i := range seqSet {
		seq := &seqSet[i]
		dyn := false
		if seq.Start == 0 {
			seq.Start = max
			dyn = true
		}
		if seq.Stop == 0 {
			seq.Stop = max
			dyn = true
		}
		if dyn && seq.Start > seq.Stop {
			seq.Start, seq.Stop = seq.Stop, seq.Start
		}
	}
}
