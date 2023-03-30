package imapserver

import (
	"fmt"
	"runtime/debug"

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

	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		defer func() {
			if v := recover(); v != nil {
				c.server.logger().Printf("panic idling: %v\n%s", v, debug.Stack())
				done <- fmt.Errorf("imapserver: panic idling")
			}
		}()
		w := &UpdateWriter{conn: c, allowExpunge: true}
		done <- c.session.Idle(w, stop)
	}()

	c.setReadTimeout(idleReadTimeout)
	line, isPrefix, err := c.br.ReadLine()
	close(stop)
	if err != nil {
		return err
	} else if isPrefix || string(line) != "DONE" {
		return newClientBugError("Syntax error: expected DONE to end IDLE command")
	}

	return <-done
}
