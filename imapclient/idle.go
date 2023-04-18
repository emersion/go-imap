package imapclient

import (
	"fmt"
)

// Idle sends an IDLE command.
//
// Unlike other commands, this method blocks until the server acknowledges it.
// On success, the IDLE command is running and other commands cannot be sent.
// The caller must invoke IdleCommand.Close to stop IDLE and unblock the
// client.
//
// This command requires support for IMAP4rev2 or the IDLE extension.
func (c *Client) Idle() (*IdleCommand, error) {
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
}

// Close stops the IDLE command.
//
// This method blocks until the command to stop IDLE is written, but doesn't
// wait for the server to respond. Callers can use Wait for this purpose.
func (cmd *IdleCommand) Close() error {
	if cmd.err != nil {
		return cmd.err
	}
	if cmd.enc == nil {
		return fmt.Errorf("imapclient: IDLE command closed twice")
	}
	cmd.enc.client.setWriteTimeout(*cmd.enc.client.options.CmdWriteTimeout)
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
	if cmd.enc != nil {
		return fmt.Errorf("imapclient: IdleCommand.Close must be called before Wait")
	}
	return cmd.cmd.Wait()
}
