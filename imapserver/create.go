package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleCreate(dec *imapwire.Decoder) error {
	var (
		name    string
		options imap.CreateOptions
	)
	if !dec.ExpectSP() || !dec.ExpectMailbox(&name) {
		return dec.Err()
	}
	if dec.SP() {
		var name string
		if !dec.ExpectSpecial('(') || !dec.ExpectAtom(&name) || !dec.ExpectSP() {
			return dec.Err()
		}
		switch strings.ToUpper(name) {
		case "USE":
			err := dec.ExpectList(func() error {
				flag, err := internal.ExpectFlag(dec)
				if err != nil {
					return err
				}
				options.SpecialUse = append(options.SpecialUse, imap.MailboxAttr(flag))
				return nil
			})
			if err != nil {
				return err
			}
		default:
			return newClientBugError("unknown CREATE parameter")
		}
		if !dec.ExpectSpecial(')') {
			return dec.Err()
		}
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}
	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}
	return c.session.Create(name, &options)
}
