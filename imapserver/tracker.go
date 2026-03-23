package imapserver

import (
	"fmt"
	"sync"

	"github.com/emersion/go-imap/v2"
)

// MailboxTracker tracks the state of a mailbox.
//
// A mailbox can have multiple sessions listening for updates. Each session has
// its own view of the mailbox, because IMAP clients asynchronously receive
// mailbox updates.
type MailboxTracker struct {
	mutex       sync.Mutex
	numMessages uint32
	sessions    map[*SessionTracker]struct{}
}

// NewMailboxTracker creates a new mailbox tracker.
func NewMailboxTracker(numMessages uint32) *MailboxTracker {
	return &MailboxTracker{
		numMessages: numMessages,
		sessions:    make(map[*SessionTracker]struct{}),
	}
}

// NewSession creates a new session tracker for the mailbox.
//
// The caller must call SessionTracker.Close once they are done with the
// session.
func (t *MailboxTracker) NewSession() *SessionTracker {
	st := &SessionTracker{mailbox: t}
	t.mutex.Lock()
	t.sessions[st] = struct{}{}
	t.mutex.Unlock()
	return st
}

func (t *MailboxTracker) queueUpdate(update *trackerUpdate, source *SessionTracker) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if update.expunge != 0 && update.expunge > t.numMessages {
		panic(fmt.Errorf("imapserver: expunge sequence number (%v) out of range (%v messages in mailbox)", update.expunge, t.numMessages))
	}
	if update.numMessages != 0 && update.numMessages < t.numMessages {
		panic(fmt.Errorf("imapserver: cannot decrease mailbox number of messages from %v to %v", t.numMessages, update.numMessages))
	}

	for st := range t.sessions {
		if source != nil && st == source {
			continue
		}
		st.queueUpdate(update)
	}

	switch {
	case update.expunge != 0:
		t.numMessages--
	case update.numMessages != 0:
		t.numMessages = update.numMessages
	}
}

// QueueExpunge queues a new EXPUNGE update.
func (t *MailboxTracker) QueueExpunge(seqNum uint32) {
	if seqNum == 0 {
		panic("imapserver: invalid expunge message sequence number")
	}
	t.queueUpdate(&trackerUpdate{expunge: seqNum}, nil)
}

// QueueNumMessages queues a new EXISTS update.
func (t *MailboxTracker) QueueNumMessages(n uint32) {
	// TODO: merge consecutive NumMessages updates
	t.queueUpdate(&trackerUpdate{numMessages: n}, nil)
}

// QueueMailboxFlags queues a new FLAGS update.
func (t *MailboxTracker) QueueMailboxFlags(flags []imap.Flag) {
	if flags == nil {
		flags = []imap.Flag{}
	}
	t.queueUpdate(&trackerUpdate{mailboxFlags: flags}, nil)
}

// QueueMessageFlags queues a new FETCH FLAGS update.
//
// If source is not nil, the update won't be dispatched to it.
func (t *MailboxTracker) QueueMessageFlags(seqNum uint32, uid imap.UID, flags []imap.Flag, source *SessionTracker) {
	t.queueUpdate(&trackerUpdate{fetch: &trackerUpdateFetch{
		seqNum: seqNum,
		uid:    uid,
		flags:  flags,
	}}, source)
}

type trackerUpdate struct {
	expunge      uint32
	numMessages  uint32
	mailboxFlags []imap.Flag
	fetch        *trackerUpdateFetch
}

type trackerUpdateFetch struct {
	seqNum uint32
	uid    imap.UID
	flags  []imap.Flag
}

// SessionTracker tracks the state of a mailbox for an IMAP client.
type SessionTracker struct {
	mailbox *MailboxTracker

	mutex   sync.Mutex
	queue   []trackerUpdate
	updates chan<- struct{}
}

// Close unregisters the session.
func (t *SessionTracker) Close() {
	t.mailbox.mutex.Lock()
	delete(t.mailbox.sessions, t)
	t.mailbox.mutex.Unlock()
	t.mailbox = nil
}

func (t *SessionTracker) queueUpdate(update *trackerUpdate) {
	var updates chan<- struct{}
	t.mutex.Lock()
	t.queue = append(t.queue, *update)
	updates = t.updates
	t.mutex.Unlock()

	if updates != nil {
		select {
		case updates <- struct{}{}:
			// we notified SessionTracker.Idle about the update
		default:
			// skip the update
		}
	}
}

// Poll dequeues pending mailbox updates for this session.
func (t *SessionTracker) Poll(w *UpdateWriter, allowExpunge bool) error {
	var updates []trackerUpdate
	t.mutex.Lock()
	if allowExpunge {
		updates = t.queue
		t.queue = nil
	} else {
		stopIndex := -1
		for i, update := range t.queue {
			if update.expunge != 0 {
				stopIndex = i
				break
			}
			updates = append(updates, update)
		}
		if stopIndex >= 0 {
			t.queue = t.queue[stopIndex:]
		} else {
			t.queue = nil
		}
	}
	t.mutex.Unlock()

	for _, update := range updates {
		var err error
		switch {
		case update.expunge != 0:
			err = w.WriteExpunge(update.expunge)
		case update.numMessages != 0:
			err = w.WriteNumMessages(update.numMessages)
		case update.mailboxFlags != nil:
			err = w.WriteMailboxFlags(update.mailboxFlags)
		case update.fetch != nil:
			err = w.WriteMessageFlags(update.fetch.seqNum, update.fetch.uid, update.fetch.flags)
		default:
			panic(fmt.Errorf("imapserver: unknown tracker update %#v", update))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Idle continuously writes mailbox updates.
//
// When the stop channel is closed, it returns.
//
// Idle cannot be invoked concurrently from two separate goroutines.
func (t *SessionTracker) Idle(w *UpdateWriter, stop <-chan struct{}) error {
	updates := make(chan struct{}, 64)
	t.mutex.Lock()
	ok := t.updates == nil
	if ok {
		t.updates = updates
	}
	t.mutex.Unlock()
	if !ok {
		return fmt.Errorf("imapserver: only a single SessionTracker.Idle call is allowed at a time")
	}

	defer func() {
		t.mutex.Lock()
		t.updates = nil
		t.mutex.Unlock()
	}()

	for {
		select {
		case <-updates:
			if err := t.Poll(w, true); err != nil {
				return err
			}
		case <-stop:
			return nil
		}
	}
}

// DecodeSeqNum converts a message sequence number from the client view to the
// server view.
//
// Zero is returned if the message doesn't exist from the server point-of-view.
func (t *SessionTracker) DecodeSeqNum(seqNum uint32) uint32 {
	if seqNum == 0 {
		return 0
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	for _, update := range t.queue {
		if update.expunge == 0 {
			continue
		}
		if seqNum == update.expunge {
			return 0
		} else if seqNum > update.expunge {
			seqNum--
		}
	}

	if seqNum > t.mailbox.numMessages {
		return 0
	}

	return seqNum
}

// EncodeSeqNum converts a message sequence number from the server view to the
// client view.
//
// Zero is returned if the message doesn't exist from the client point-of-view.
func (t *SessionTracker) EncodeSeqNum(seqNum uint32) uint32 {
	if seqNum == 0 {
		return 0
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	if seqNum > t.mailbox.numMessages {
		return 0
	}

	for i := len(t.queue) - 1; i >= 0; i-- {
		update := t.queue[i]
		// TODO: this doesn't handle increments > 1
		if update.numMessages != 0 && seqNum == update.numMessages {
			return 0
		}
		if update.expunge != 0 && seqNum >= update.expunge {
			seqNum++
		}
	}
	return seqNum
}
