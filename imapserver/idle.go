package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleIdle(dec *imapwire.Decoder) error {
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	if err := c.writeContReq("idling"); err != nil {
		return err
	}

	c.setReadTimeout(idleReadTimeout)
	line, isPrefix, err := c.br.ReadLine()
	if err != nil {
		return err
	} else if isPrefix || string(line) != "DONE" {
		return newClientBugError("Syntax error: expected DONE to end IDLE command")
	}

	return nil
}
