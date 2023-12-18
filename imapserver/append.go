package imapserver

import (
	"fmt"
	"io"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// appendLimit is the maximum size of an APPEND payload.
//
// TODO: make configurable
const appendLimit = 100 * 1024 * 1024 // 100MiB

func (c *Conn) handleAppend(tag string, dec *imapwire.Decoder) error {
	var (
		mailbox string
		options imap.AppendOptions
	)
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectSP() {
		return dec.Err()
	}

	hasFlagList, err := dec.List(func() error {
		flag, err := internal.ExpectFlag(dec)
		if err != nil {
			return err
		}
		options.Flags = append(options.Flags, flag)
		return nil
	})
	if err != nil {
		return err
	}
	if hasFlagList && !dec.ExpectSP() {
		return dec.Err()
	}

	t, err := internal.DecodeDateTime(dec)
	if err != nil {
		return err
	}
	if !t.IsZero() && !dec.ExpectSP() {
		return dec.Err()
	}
	options.Time = t

	lit, nonSync, err := dec.ExpectLiteralReader()
	if err != nil {
		return err
	}
	if lit.Size() > appendLimit {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeTooBig,
			Text: fmt.Sprintf("Literals are limited to %v bytes for this command", appendLimit),
		}
	}
	if err := c.acceptLiteral(lit.Size(), nonSync); err != nil {
		return err
	}

	c.setReadTimeout(literalReadTimeout)
	defer c.setReadTimeout(cmdReadTimeout)

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		io.Copy(io.Discard, lit)
		dec.CRLF()
		return err
	}

	data, appendErr := c.session.Append(mailbox, lit, &options)
	if _, discardErr := io.Copy(io.Discard, lit); discardErr != nil {
		return err
	}
	if !dec.ExpectCRLF() {
		return err
	}
	if appendErr != nil {
		return appendErr
	}
	if err := c.poll("APPEND"); err != nil {
		return err
	}
	return c.writeAppendOK(tag, data)
}

func (c *Conn) writeAppendOK(tag string, data *imap.AppendData) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom(tag).SP().Atom("OK").SP()
	if data != nil {
		enc.Special('[')
		enc.Atom("APPENDUID").SP().Number(data.UIDValidity).SP().UID(data.UID)
		enc.Special(']').SP()
	}
	enc.Text("APPEND completed")
	return enc.CRLF()
}
