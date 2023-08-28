package imapclient

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
)

const idleFallbackInterval = time.Minute

// Idle sends an IDLE command.
//
// Unlike other commands, this method blocks until the server acknowledges it.
// On success, the IDLE command is running and other commands cannot be sent.
// The caller must invoke IdleCommand.Close to stop IDLE and unblock the
// client.
//
// If the server lacks support for IMAP4rev2 or IDLE, this function falls back
// to polling based on NOOP.
func (c *Client) Idle() (*IdleCommand, error) {
	if !c.Caps().Has(imap.CapIdle) {
		if err := c.Noop().Wait(); err != nil {
			return nil, err
		}
		return &IdleCommand{fallback: newIdleCommandFallback(c)}, nil
	}

	cmd := &IdleCommand{}
	contReq := c.registerContReq(cmd)
	cmd.enc = c.beginCommand("IDLE", cmd)
	cmd.enc.flush()

	_, err := contReq.Wait()
	if err != nil {
		cmd.enc.end()
		return nil, err
	}

	return cmd, nil
}

// IdleCommand is an IDLE command.
//
// Initially, the IDLE command is running. The server may send unilateral
// data. The client cannot send any command while IDLE is running.
//
// Close must be called to stop the IDLE command.
type IdleCommand struct {
	cmd
	enc *commandEncoder

	fallback *idleCommandFallback
}

// Close stops the IDLE command.
//
// This method blocks until the command to stop IDLE is written, but doesn't
// wait for the server to respond. Callers can use Wait for this purpose.
func (cmd *IdleCommand) Close() error {
	if cmd.fallback != nil {
		close(cmd.fallback.stop)
		return nil
	}

	if cmd.err != nil {
		return cmd.err
	}
	if cmd.enc == nil {
		return fmt.Errorf("imapclient: IDLE command closed twice")
	}
	cmd.enc.client.setWriteTimeout(cmdWriteTimeout)
	_, err := cmd.enc.client.bw.WriteString("DONE\r\n")
	if err == nil {
		err = cmd.enc.client.bw.Flush()
	}
	cmd.enc.end()
	cmd.enc = nil
	return err
}

// Wait blocks until the IDLE command has completed.
//
// Wait can only be called after Close.
func (cmd *IdleCommand) Wait() error {
	if cmd.fallback != nil {
		<-cmd.fallback.done
		return cmd.fallback.err
	}

	if cmd.enc != nil {
		return fmt.Errorf("imapclient: IdleCommand.Close must be called before Wait")
	}
	return cmd.cmd.Wait()
}

type idleCommandFallback struct {
	client *Client
	timer  *time.Timer
	stop   chan struct{}
	done   chan struct{}
	err    error
}

func newIdleCommandFallback(c *Client) *idleCommandFallback {
	return &idleCommandFallback{
		client: c,
		timer:  time.NewTimer(idleFallbackInterval),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (fallback *idleCommandFallback) run() {
	defer close(fallback.done)
	defer fallback.timer.Stop()

	for {
		select {
		case <-fallback.timer.C:
			if err := fallback.client.Noop().Wait(); err != nil {
				fallback.err = err
				return
			}
			fallback.timer.Reset(idleFallbackInterval)
		case <-fallback.stop:
			return
		}
	}
}
