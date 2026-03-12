package imapserver

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// ThreadData represents a single thread in a THREAD response.
type ThreadData struct {
	Chain      []uint32
	SubThreads []ThreadData
}

// SessionThread is an IMAP session which supports THREAD.
type SessionThread interface {
	Session

	// Thread returns message threads matching the given criteria using the
	// specified algorithm. numKind indicates whether to return sequence
	// numbers or UIDs.
	Thread(numKind NumKind, algorithm imap.ThreadAlgorithm, criteria *imap.SearchCriteria) ([]ThreadData, error)
}

func (c *Conn) handleThread(tag string, dec *imapwire.Decoder, numKind NumKind) error {
	if !dec.ExpectSP() {
		return dec.Err()
	}

	// Parse algorithm
	var algorithm string
	if !dec.ExpectAtom(&algorithm) || !dec.ExpectSP() {
		return dec.Err()
	}

	// Parse charset
	var charset string
	if !dec.ExpectAtom(&charset) || !dec.ExpectSP() {
		return dec.Err()
	}
	if !strings.EqualFold(charset, "UTF-8") && !strings.EqualFold(charset, "US-ASCII") {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeBadCharset,
			Text: "Only UTF-8 and US-ASCII are supported for THREAD",
		}
	}

	// Parse search criteria
	var criteria imap.SearchCriteria
	for {
		if err := readSearchKey(&criteria, dec); err != nil {
			return fmt.Errorf("in search-key: %w", err)
		}
		if !dec.SP() {
			break
		}
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	threadSession, ok := c.session.(SessionThread)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeCannot,
			Text: "THREAD command is not supported by this session",
		}
	}

	threads, err := threadSession.Thread(numKind, imap.ThreadAlgorithm(strings.ToUpper(algorithm)), &criteria)
	if err != nil {
		return err
	}

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("THREAD")
	for _, thread := range threads {
		enc.SP()
		writeThread(enc.Encoder, &thread)
	}
	return enc.CRLF()
}

func writeThread(enc *imapwire.Encoder, thread *ThreadData) {
	enc.Special('(')
	hasContent := false
	for i, num := range thread.Chain {
		if i > 0 {
			enc.SP()
		}
		enc.Number(num)
		hasContent = true
	}
	for i, sub := range thread.SubThreads {
		// Fix #5: Always insert separator between items, whether chain
		// is empty or not.
		if hasContent || i > 0 {
			enc.SP()
		}
		writeThread(enc, &sub)
	}
	enc.Special(')')
}
